package process

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

type Executor interface {
	Execute(operationID string) (time.Duration, error)
}

type Queue struct {
	queue     workqueue.RateLimitingInterface
	executor  Executor
	waitGroup sync.WaitGroup
	log       logrus.FieldLogger
}

func NewQueue(executor Executor, log logrus.FieldLogger) *Queue {
	return &Queue{
		queue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "operations"),
		executor:  executor,
		waitGroup: sync.WaitGroup{},
		log:       log,
	}
}

func (q *Queue) Add(processId string) {
	q.queue.Add(processId)
}

func (q *Queue) ShutDown() {
	q.queue.ShutDown()
}

func (q *Queue) Run(stop <-chan struct{}, workersAmount int) {
	for i := 0; i < workersAmount; i++ {
		q.waitGroup.Add(1)
		createWorker(q.queue, q.executor.Execute, stop, &q.waitGroup, q.log)
	}
}

func createWorker(queue workqueue.RateLimitingInterface, process func(id string) (time.Duration, error), stopCh <-chan struct{}, waitGroup *sync.WaitGroup, log logrus.FieldLogger) {
	go func() {
		wait.Until(worker(queue, process, log), time.Second, stopCh)
		waitGroup.Done()
	}()
}

func worker(queue workqueue.RateLimitingInterface, process func(key string) (time.Duration, error), log logrus.FieldLogger) func() {
	return func() {
		exit := false
		for !exit {
			exit = func() bool {
				key, quit := queue.Get()
				if quit {
					return true
				}
				id := key.(string)
				log = log.WithField("operationID", id)
				defer func() {
					if err := recover(); err != nil {
						log.Errorf("panic error from process: %v", err)
					}
					queue.Done(key)
				}()

				when, err := process(id)
				if err == nil && when != 0 {
					log.Infof("Adding %q item after %s", id, when)
					queue.AddAfter(key, when)
					return false
				}
				if err != nil {
					log.Errorf("Error from process: %v", err)
				}

				queue.Forget(key)
				return false
			}()
		}
	}
}
