package process

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

const (
	workersAmount = 5
)

type Executor interface {
	Execute(operationID string) (time.Duration, error)
}

type Queue struct {
	queue    workqueue.RateLimitingInterface
	executor Executor
	log      logrus.FieldLogger
}

func NewQueue(executor Executor, log logrus.FieldLogger) *Queue {
	return &Queue{
		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "operations"),
		executor: executor,
		log:      log,
	}
}

func (q *Queue) Add(processId string) {
	q.queue.Add(processId)
}

func (q *Queue) Run(stop <-chan struct{}) {
	var waitGroup sync.WaitGroup

	for i := 0; i < workersAmount; i++ {
		createWorker(q.queue, q.executor.Execute, stop, &waitGroup, q.log)
	}
}

func createWorker(queue workqueue.RateLimitingInterface, process func(id string) (time.Duration, error), stopCh <-chan struct{}, waitGroup *sync.WaitGroup, log logrus.FieldLogger) {
	waitGroup.Add(1)
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
				log = log.WithField("operationID", key)
				defer func() {
					if err := recover(); err != nil {
						log.Errorf("panic error from process: %v", err)
					}
					queue.Done(key)
				}()

				when, err := process(key.(string))
				if err == nil && when != 0 {
					log.Infof("Adding %q item after %s", key.(string), when)
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
