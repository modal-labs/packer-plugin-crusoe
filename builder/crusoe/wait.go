package crusoe

import (
	"fmt"
	"log"
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

// waitForImageState waits until the image reaches the desired state
func waitForImageState(state, imageID string, client *Client, timeout time.Duration) error {
	done := make(chan struct{})
	defer close(done)
	result := make(chan error, 1)

	go func() {
		attempts := 0
		for {
			attempts++
			log.Printf("Checking image status... (attempt: %d)", attempts)

			image, err := client.GetCustomImage(imageID)
			if err != nil {
				result <- err
				return
			}

			if image.State == state {
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

	log.Printf("Waiting for up to %d seconds for image", timeout/time.Second)
	select {
	case err := <-result:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timeout while waiting for image")
	}
}
