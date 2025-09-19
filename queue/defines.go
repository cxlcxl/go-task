package queue

import "github.com/panjf2000/ants/v2"

var (
// river.AddWorker(workers, &DemoWorker{manager: manager})
// river.AddWorker(workers, &EmailWorker{manager: manager})
// river.AddWorker(workers, &DataSyncWorker{manager: manager})
// river.AddWorker(workers, &FileProcessWorker{manager: manager})
// river.AddWorker(workers, &ReportGenerateWorker{manager: manager})
// river.AddWorker(workers, &NotificationWorker{manager: manager})
// river.AddWorker(workers, &DemoWorker{manager: manager})
)

func dd() {
	pool, _ := ants.NewPool(100)
	//pool.Submit()
	pool.Cap()
	pool.Free()
}
