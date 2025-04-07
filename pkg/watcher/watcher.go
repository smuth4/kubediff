package watcher

import (
	"time"

	"github.com/arriqaaq/kubediff/config"
	"github.com/arriqaaq/kubediff/pkg/log"
	"github.com/arriqaaq/kubediff/pkg/notify"
)

const (
	resyncPeriod = time.Duration(1) * time.Minute
)

func NewWatcher(cfg *config.Config) (*Watcher, error) {
	kubeconfig, err := newKubeConfig()
	if err != nil {
		return nil, err
	}

	client, err := newClient(kubeconfig)
	if err != nil {
		return nil, err
	}

	informer, err := NewMultiResourceInformer(cfg, resyncPeriod)(client)
	if err != nil {
		return nil, err
	}

	return &Watcher{client, informer, cfg}, nil
}

type Watcher struct {
	client   *Client
	informer Informer
	cfg      *config.Config
}

func (w *Watcher) Run(stopCh chan struct{}) {
	go w.informer.Start(stopCh)
	notifier := notify.NewNotifierList(w.cfg)
	w.informer.AddEventHandler(getEventHandler(w.cfg), notifier)
	log.Debug("waiting for informer cache sync")
	w.informer.WaitForCacheSync(stopCh)
	HasSyncedMutex.Lock()
	HasSynced = true
	HasSyncedMutex.Unlock()
	log.Debug("cache sync complete")
	<-stopCh
}
