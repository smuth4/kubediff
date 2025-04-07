package watcher

import (
	"strings"
	"sync"

	"github.com/arriqaaq/kubediff/config"
	"github.com/arriqaaq/kubediff/pkg/event"
	"github.com/arriqaaq/kubediff/pkg/log"
	"github.com/arriqaaq/kubediff/pkg/notify"
	"github.com/r3labs/diff/v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
)

const (
	EventAdd    string = "EventAdd"
	EventUpdate string = "EventUpdate"
	EventDelete string = "EventDelete"
)

// Bit of a gross hack, but we can ensure there's only one informer manipulating this
var HasSynced bool
var HasSyncedMutex sync.Mutex

type eventHandler func(resourceType string, notifier notify.Notifier) cache.ResourceEventHandlerFuncs

func watchHandler(resourceType string, notifier notify.Notifier) cache.ResourceEventHandlerFuncs {
	var handler cache.ResourceEventHandlerFuncs
	handler.AddFunc = func(obj interface{}) {
		log.WithField("resourceType", resourceType).WithField("obj", obj).Info("add event")
		notifier.Handle(event.NewEvent(EventAdd, resourceType, obj, nil))
	}
	handler.UpdateFunc = func(old, new interface{}) {
		log.WithField("resourceType", resourceType).WithField("old", old).WithField("new", new).Info("update event")
		notifier.Handle(event.NewEvent(EventUpdate, resourceType, new, nil))
	}
	handler.DeleteFunc = func(obj interface{}) {
		log.WithField("resourceType", resourceType).WithField("obj", obj).Info("delete event")
		notifier.Handle(event.NewEvent(EventDelete, resourceType, obj, nil))
	}
	return handler
}

func hasSynced() bool {
	HasSyncedMutex.Lock()
	defer HasSyncedMutex.Unlock()
	return HasSynced
}

func diffHandlerFactory(cfg *config.Config) func(resourceType string, notifier notify.Notifier) cache.ResourceEventHandlerFuncs {
	return func(resourceType string, notifier notify.Notifier) cache.ResourceEventHandlerFuncs {
		var handler cache.ResourceEventHandlerFuncs
		handler.AddFunc = func(obj interface{}) {
			if !hasSynced() {
				return
			}
			objStruct := obj.(*unstructured.Unstructured)
			addLog := log.WithField("name", objStruct.GetName()).WithField("namespace", objStruct.GetNamespace())
			addLog.WithField("resourceType", resourceType).Info("add event")
			notifier.Handle(event.NewEvent(EventAdd, resourceType, obj, nil))
		}
		handler.UpdateFunc = func(old, new interface{}) {
			oldObj := old.(*unstructured.Unstructured)
			newObj := new.(*unstructured.Unstructured)

			diff, _ := diff.Diff(oldObj, newObj)
			if len(diff) == 0 {
				return
			}

			// Adapted from r3labs/diff to make Path a string
			type Change struct {
				Type string      `json:"type"`
				Path string      `json:"path"`
				From interface{} `json:"from"`
				To   interface{} `json:"to"`
			}

			changes := []Change{}
			for _, d := range diff {
				path := strings.Join(d.Path[1:], "/")
				if cfg.IsIgnoredDiffPath(resourceType, path) {
					continue
				}
				changes = append(changes, Change{
					Type: d.Type,
					Path: path,
					From: d.From,
					To:   d.To,
				})
			}
			if len(changes) == 0 {
				return
			}
			diffLog := log.WithField("name", newObj.GetName()).WithField("namespace", newObj.GetNamespace())
			diffLog = diffLog.WithField("resourceType", resourceType).WithField("diff", changes)
			diffLog.Info("update event")
			notifier.Handle(event.NewEvent(EventUpdate, resourceType, old, diff))
		}
		handler.DeleteFunc = func(obj interface{}) {
			objStruct := obj.(*unstructured.Unstructured)
			addLog := log.WithField("name", objStruct.GetName()).WithField("namespace", objStruct.GetNamespace())
			addLog.WithField("resourceType", resourceType).Info("delete event")
			notifier.Handle(event.NewEvent(EventAdd, resourceType, obj, nil))
		}
		return handler
	}
}

func noOpHandler(resourceType string, notifier notify.Notifier) cache.ResourceEventHandlerFuncs {

	var handler cache.ResourceEventHandlerFuncs
	handler.AddFunc = func(obj interface{}) {
		log.WithField("resourceType", resourceType).Info("delete event")
	}
	handler.UpdateFunc = func(old, new interface{}) {
		log.WithField("resourceType", resourceType).Info("delete event")
	}
	handler.DeleteFunc = func(obj interface{}) {
		log.WithField("resourceType", resourceType).Info("delete event")
	}
	return handler
}

func getEventHandler(cfg *config.Config) eventHandler {
	switch cfg.Mode {
	case config.DiffMode:
		return diffHandlerFactory(cfg)
	default:
		return watchHandler
	}
}
