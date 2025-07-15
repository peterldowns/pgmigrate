package sessionlock

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for postgres
	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgmigrate/internal/withdb"
)

func TestWithSessionLock(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	check.Nil(t, withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		var counter int32
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()
				err := With(ctx, db, "test-with-session-lock", func(_ *sql.Conn) error {
					newCounter := atomic.AddInt32(&counter, 1)
					check.Equal(t, int32(1), newCounter)

					time.Sleep(time.Millisecond * 10)

					newCounter = atomic.AddInt32(&counter, -1)
					check.Equal(t, int32(0), newCounter)

					return nil
				})

				check.Nil(t, err)
			}()
		}
		wg.Wait()
		return nil
	}))
}

func TestWithReturnsErrorsFromCallback(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	check.Nil(t, withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		// This error should be the same error returned by conn
		err := With(ctx, db, "example", func(conn *sql.Conn) error {
			_, err := conn.ExecContext(ctx, "select broken query")
			return err
		})
		check.NotEqual(t, nil, err)
		return nil
	}))
}

func TestWithReturnsUnlockErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	check.Nil(t, withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		err := With(ctx, db, "example", func(conn *sql.Conn) error {
			err := conn.Close()
			if !check.Nil(t, err) {
				return fmt.Errorf("inner: %w", err)
			}
			return nil
		})
		assert.NotEqual(t, nil, err)
		msgs := strings.Split(err.Error(), "\n")
		check.Equal(t, []string{
			"sessionlock(example) failed to unlock: sql: connection is already closed",
			"sessionlock(example) failed to close conn: sql: connection is already closed",
		}, msgs)
		return nil
	}))
}

func TestWithSucceedsDespiteSessionLockAndStatementTimeouts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	check.Nil(t, withdb.WithDBParams(ctx, "pgx", "", func(db *sql.DB) error {
		// Make it so that any statement that waits on a lock for 50ms causes
		// immediate failure.  Similarly, any statement taking 51ms will cause
		// immediate failure.
		if _, err := db.ExecContext(ctx, "SET lock_timeout = '50ms'"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "SET statement_timeout = '51ms'"); err != nil {
			return err
		}
		lockName := "test-with-session-lock-spins"
		errCh := make(chan error)
		ackCh := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := With(ctx, db, lockName, func(_ *sql.Conn) error {
				// (1) This callback executes once the lock is acquired; writing to
				// this channel will trigger another goroutine to try to acquire
				// the same lock.
				ackCh <- struct{}{}
				// Wait for longer than the lock_timeout and statement_timeout
				// before relinquishing the session lock.
				//
				// This will test that the other goroutine waits without
				// triggering those timeouts. If it uses pg_advisory_lock, it
				// will end up waiting long enough to cause a timeout, and the
				// test fails. If it uses pg_try_advisory_lock and a spin loop,
				// it won't cause a lock or statement timeout, and the test will
				// pass.
				time.Sleep(200 * time.Millisecond)
				return nil
			})
			if err != nil {
				errCh <- err
			}
		}()

		// Wait up to 400ms for the first session lock to be acquired, as
		// signaled by (1) in the first goroutine. If there's an error or it
		// takes longer than that, fail the test immediately.
		assert.Nil(t, waitForAcquired(errCh, ackCh, 400*time.Millisecond))

		// This goroutine should spin a few times while waiting to acquire the
		// lock, then successfully acquire it. The time it waits to acquire the
		// lock should be longer than the configured lock_timeout and
		// statement_timeout parameters, but that's OK because [With] spins and
		// uses `pg_try_advisory_lock` so there are no
		// long-running/hanging/blocked lock acquisition statements.
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := With(ctx, db, lockName, func(_ *sql.Conn) error {
				// (2) Signal that the session lock was acquired successfully by
				// this second goroutine.
				ackCh <- struct{}{}
				return nil
			})
			if err != nil {
				errCh <- err
			}
		}()

		// Wait up to 400ms for the session lock to be acquired again, as
		// signaled by (2) in the second goroutine. It should take ~200ms based
		// on the `time.Sleep` in the first goroutine. If this hangs for longer
		// than 400ms or there's an error, fail the test immediately.
		assert.Nil(t, waitForAcquired(errCh, ackCh, 400*time.Millisecond))

		// Wait for all locks to be released.
		wg.Wait()
		return nil
	}))
}

func waitForAcquired(errch chan error, lockch chan struct{}, timeout time.Duration) error {
	select {
	case err := <-errch:
		return err
	case <-lockch:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timed out waiting for lock acquisition")
	}
}
