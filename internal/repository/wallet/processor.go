package wallet

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Black1black/go_base_api/internal/models"
	"github.com/google/uuid"
)

type Operation struct {
	WalletID      uuid.UUID
	Amount        int64
	OperationType models.OperationType
	ResultChan    chan error
	Context       context.Context
}

type OperationProcessor struct {
	repo   *Repository
	queue  chan *Operation
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewOperationProcessor(repo *Repository) *OperationProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	const workers = 20
	const queueSize = 10000

	processor := &OperationProcessor{
		repo:   repo,
		queue:  make(chan *Operation, queueSize),
		ctx:    ctx,
		cancel: cancel,
	}

	for i := 0; i < workers; i++ {
		processor.wg.Add(1)
		go processor.worker(i)
	}

	log.Printf("✅ OperationProcessor started with %d workers, queue size: %d", workers, queueSize)

	return processor
}

func (p *OperationProcessor) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			log.Printf("Worker %d stopping", id)
			return
		case op := <-p.queue:
			p.processOperation(op, id)
		}
	}
}

func (p *OperationProcessor) processOperation(op *Operation, workerID int) {
	for attempt := 0; ; attempt++ {
		select {
		case <-op.Context.Done():
			op.ResultChan <- op.Context.Err()
			return
		case <-p.ctx.Done():
			op.ResultChan <- p.ctx.Err()
			return
		default:
		}

		err := p.repo.updateBalanceTx(op.Context, op.WalletID, op.Amount, op.OperationType)

		if err == nil {
			if attempt > 0 {
				log.Printf("Worker %d: ✅ Operation succeeded after %d retries for wallet %s",
					workerID, attempt, op.WalletID)
			}
			op.ResultChan <- nil
			return
		}

		if attempt%10 == 0 {
			log.Printf("Worker %d: ⏳ Retrying operation for wallet %s (attempt %d): %v",
				workerID, op.WalletID, attempt+1, err)
		}

		backoff := time.Duration(10*(1<<min(attempt, 7))) * time.Millisecond
		if backoff > time.Second {
			backoff = time.Second
		}

		select {
		case <-time.After(backoff):
		case <-op.Context.Done():
			op.ResultChan <- op.Context.Err()
			return
		case <-p.ctx.Done():
			op.ResultChan <- p.ctx.Err()
			return
		}
	}
}

func (p *OperationProcessor) Process(ctx context.Context, walletID uuid.UUID,
	amount int64, operationType models.OperationType) error {

	resultChan := make(chan error, 1)

	op := &Operation{
		WalletID:      walletID,
		Amount:        amount,
		OperationType: operationType,
		ResultChan:    resultChan,
		Context:       ctx,
	}

	select {
	case p.queue <- op:
		select {
		case err := <-resultChan:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *OperationProcessor) Shutdown() {
	log.Println("🛑 Shutting down OperationProcessor...")
	p.cancel()
	p.wg.Wait()
	close(p.queue)
	log.Println("✅ OperationProcessor stopped")
}
