package crusoe

import (
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	sleepDurationSeconds = 3
)

// waitForInstanceState waits until the instance reaches the desired state
func waitForInstanceState(state, instanceID string, client *Client, timeout time.Duration) error {
	done := make(chan struct{})
	defer close(done)
	result := make(chan error, 1)

	go func() {
		attempts := 0
		for {
			attempts++
			log.Printf("Checking instance status... (attempt: %d)", attempts)

			instance, err := client.GetInstance(instanceID)
			if err != nil {
				// Handle 404 errors during the initial creation period
				// The instance may not exist immediately after creation starts
				if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not_found") {
					log.Printf("Instance not found yet, will retry... (attempt: %d)", attempts)
					time.Sleep(sleepDurationSeconds * time.Second)
					// Verify we shouldn't exit
					select {
					case <-done:
						return
					default:
						continue
					}
				}
				// For other errors, fail immediately
				result <- err
				return
			}

			if instance.State == state {
				result <- nil
				return
			}

			time.Sleep(sleepDurationSeconds * time.Second)

			// Verify we shouldn't exit
			select {
			case <-done:
				return
			default:
			}
		}
	}()

	log.Printf("Waiting for up to %d seconds for instance", timeout/time.Second)
	select {
	case err := <-result:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timeout while waiting for instance")
	}
}
