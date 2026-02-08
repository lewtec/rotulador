package annotation

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
)

// HandleHome serves the home page
func (a *AnnotatorApp) HandleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}

	data := map[string]interface{}{
		"Title":       "Welcome to Rotulador",
		"ProjectName": "Welcome to Rotulador",
		"Description": a.Config.Meta.Description,
	}

	err := RenderPageWithRequest(r, w, "home.html", data)
	if err != nil {
		a.Logger.Error("error rendering home template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// HandleFavicon serves the favicon
func (a *AnnotatorApp) HandleFavicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	if _, err := w.Write([]byte(GetFavicon())); err != nil {
		a.Logger.Error("error writing favicon response", "err", err)
	}
}

// HandleHelp serves the help pages
func (a *AnnotatorApp) HandleHelp(w http.ResponseWriter, r *http.Request) {
	itemPath := pathParts(r.URL.Path)
	title := "Help"

	var tasks []TaskWithCount
	var currentTask *ConfigTask

	if len(itemPath) == 1 {
		// Only populate tasks for the timeline view (no markdown for tasks)
		tasks = make([]TaskWithCount, 0, len(a.Config.Tasks))

		for _, task := range a.Config.Tasks {
			availableCount, err := a.CountAvailableImages(r.Context(), task.ID)
			if err != nil {
				a.Logger.Error("error counting available images", "task", task.ID, "err", err)
				availableCount = 0
			}

			totalEligible, err := a.CountEligibleImages(r.Context(), task.ID)
			if err != nil {
				a.Logger.Error("error counting eligible images", "task", task.ID, "err", err)
				totalEligible = availableCount // fallback to available
			}

			completedCount := totalEligible - availableCount
			if completedCount < 0 {
				completedCount = 0
			}

			// Get comprehensive phase progress stats
			phaseProgress, err := a.GetPhaseProgressStats(r.Context(), task.ID)
			if err != nil {
				a.Logger.Error("error getting phase progress", "task", task.ID, "err", err)
				phaseProgress = &PhaseProgress{}
			}

			tasks = append(tasks, TaskWithCount{
				ConfigTask:     task,
				AvailableCount: availableCount,
				TotalCount:     totalEligible,
				CompletedCount: completedCount,
				PhaseProgress:  phaseProgress,
			})
		}
	} else if len(itemPath) == 2 {
		helpTask := itemPath[1]
		task := a.GetTask(helpTask)
		if task == nil {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		currentTask = task

		// Get progress stats for this specific task
		phaseProgress, err := a.GetPhaseProgressStats(r.Context(), helpTask)
		if err != nil {
			a.Logger.Error("error getting phase progress", "task", helpTask, "err", err)
			phaseProgress = &PhaseProgress{}
		}

		// Get available count to check if there are images to annotate
		availableCount, err := a.CountAvailableImages(r.Context(), helpTask)
		if err != nil {
			a.Logger.Error("error counting available images", "task", helpTask, "err", err)
			availableCount = 0
		}

		tasks = []TaskWithCount{
			{
				ConfigTask:     task,
				AvailableCount: availableCount,
				TotalCount:     phaseProgress.Completed + phaseProgress.Pending,
				CompletedCount: phaseProgress.Completed,
				PhaseProgress:  phaseProgress,
			},
		}
	} else {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}

	data := map[string]interface{}{
		"Title":       title,
		"Description": a.Config.Meta.Description,
		"Task":        currentTask,
		"Tasks":       tasks,
	}

	err := RenderPageWithRequest(r, w, "help.html", data)
	if err != nil {
		a.Logger.Error("error rendering help template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// HandleAnnotate serves the annotation interface and handles submissions
func (a *AnnotatorApp) HandleAnnotate(w http.ResponseWriter, r *http.Request) {
	itemPath := pathParts(r.URL.Path)

	if len(itemPath) != 3 {
		taskID := r.URL.Query().Get("task")
		step, err := a.NextAnnotationStep(r.Context(), taskID)
		if err != nil {
			a.Logger.Error("error in annotate when getting next step from scratch", "err", err)
			w.WriteHeader(500)
			return
		}
		if step == nil {
			data := map[string]interface{}{
				"Title": "All annotations are done!",
			}
			err := RenderPageWithRequest(r, w, "complete.html", data)
			if err != nil {
				a.Logger.Error("error rendering complete template", "err", err)
			}
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/annotate/%s/%s", step.TaskID, step.ImageID), http.StatusSeeOther)
		return
	}

	taskID := itemPath[1]
	imageID := itemPath[2]
	task := a.GetTask(taskID)
	if task == nil {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	imageFilename, _ := a.GetImageFilename(r.Context(), imageID)

	if r.Method == http.MethodPost {
		a.Logger.Debug("POST")
		if err := r.ParseForm(); err != nil {
			a.Logger.Error("failed to parse form", "err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !r.Form.Has("selectedClass") || !r.Form.Has("sure") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		selectedClass := r.FormValue("selectedClass")
		_, isClassValid := task.Classes[selectedClass]
		a.Logger.Debug("Selected class", "class", selectedClass, "empty", selectedClass == "", "valid", isClassValid)
		sure := r.FormValue("sure") == "on"
		a.Logger.Debug("Sure", "sure", sure)
		user, _, _ := r.BasicAuth()
		err := a.SubmitAnnotation(r.Context(), AnnotationResponse{
			ImageID: imageID,
			TaskID:  taskID,
			User:    user,
			Value:   selectedClass,
			Sure:    sure,
		})
		if err != nil {
			a.Logger.Error("error while submitting annotation", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		step, err := a.NextAnnotationStep(r.Context(), taskID)
		if err != nil {
			a.Logger.Error("error while getting next step", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if step == nil {
			step, err = a.NextAnnotationStep(r.Context(), "")
			if err != nil {
				a.Logger.Error("error while getting next step at the end of task", "err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		if step == nil {
			w.Header().Add("HX-Redirect", "/")
		} else if step.TaskID != taskID {
			w.Header().Add("HX-Redirect", fmt.Sprintf("/help/%s", step.TaskID))
		} else {
			w.Header().Add("HX-Redirect", fmt.Sprintf("/annotate/%s/%s", taskID, step.ImageID))
		}
		return
	}

	// Build classes with keyboard shortcuts
	classNames := make([]string, 0, len(task.Classes))
	for class := range task.Classes {
		classNames = append(classNames, class)
	}
	sort.Strings(classNames)

	classes := []ClassButton{}
	keyIndex := 1
	for _, className := range classNames {
		classMeta := task.Classes[className]
		key := ""
		if keyIndex <= 9 {
			key = fmt.Sprintf("%d", keyIndex)
			keyIndex++
		}
		classes = append(classes, ClassButton{
			ID:   className,
			Name: i(classMeta.Name),
			Key:  key,
		})
	}

	// Get comprehensive progress information
	phaseProgress, err := a.GetPhaseProgressStats(r.Context(), taskID)
	if err != nil {
		a.Logger.Error("error getting phase progress", "err", err)
		// Fallback to empty progress
		phaseProgress = &PhaseProgress{}
	}

	data := map[string]interface{}{
		"Title":         "annotation",
		"TaskID":        taskID,
		"TaskName":      task.Name,
		"ImageID":       imageID,
		"ImageFilename": imageFilename,
		"Classes":       classes,
		"PhaseProgress": phaseProgress,
		// Keep old Progress for backward compatibility
		"Progress": map[string]interface{}{
			"AvailableCount": phaseProgress.Pending,
			"TotalCount":     phaseProgress.Completed + phaseProgress.Pending,
			"CompletedCount": phaseProgress.Completed,
		},
	}

	err = RenderPageWithRequest(r, w, "annotate.html", data)
	if err != nil {
		a.Logger.Error("error rendering annotate template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// HandleAsset serves image assets
func (a *AnnotatorApp) HandleAsset(w http.ResponseWriter, r *http.Request) {
	itemPath := pathParts(r.URL.Path)
	if len(itemPath) != 2 {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	sha256 := itemPath[1]
	a.Logger.Debug("http: fetching asset", "sha256", sha256)

	// Get image filename from repository
	filename, err := a.GetImageFilename(r.Context(), sha256)
	if err != nil {
		a.Logger.Warn("http: asset was not found", "sha256", sha256, "err", err)
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}

	a.Logger.Debug("http: asset is", "sha256", sha256, "filename", filename)
	fullPath, err := secureJoin(a.ImagesDir, filename)
	if err != nil {
		a.Logger.Warn("http: asset path security check failed", "sha256", sha256, "filename", filename, "err", err)
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	f, err := os.Open(fullPath)
	if errors.Is(err, os.ErrNotExist) {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		a.Logger.Error("error: http: while serving image asset", "err", err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			ReportError(r.Context(), err, "msg", "failed to close asset file")
		}
	}()
	if _, err := io.Copy(w, f); err != nil {
		a.Logger.Error("error: http: while copying image asset", "err", err)
	}
}
