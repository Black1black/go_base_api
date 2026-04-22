package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

type TestConfig struct {
	BaseURL    string
	APIHost    string
	APIPort    string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

func LoadTestConfig(t *testing.T) *TestConfig {
	apiHost := getEnv("TEST_API_HOST", "localhost")
	apiPort := getEnv("TEST_API_PORT", "8080")

	if os.Getenv("IN_DOCKER") == "true" {
		apiHost = getEnv("TEST_API_HOST", "test-api")
	}

	dbHost := getEnv("TEST_DB_HOST", "localhost")
	dbPort := getEnv("TEST_DB_PORT", "5432")

	if os.Getenv("IN_DOCKER") == "true" {
		dbHost = getEnv("TEST_DB_HOST", "test-db")
	}

	cfg := &TestConfig{
		BaseURL:    fmt.Sprintf("http://%s:%s", apiHost, apiPort),
		APIHost:    apiHost,
		APIPort:    apiPort,
		DBHost:     dbHost,
		DBPort:     dbPort,
		DBUser:     getEnv("TEST_DB_USER", "postgres"),
		DBPassword: getEnv("TEST_DB_PASSWORD", "postgres"),
		DBName:     getEnv("TEST_DB_NAME", "wallet_test"),
	}

	t.Logf("Test configuration: API URL=%s, DB Host=%s", cfg.BaseURL, cfg.DBHost)
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (tc *TestConfig) getAPIURL(path string) string {
	return tc.BaseURL + path
}

func waitForAPI(t *testing.T, cfg *TestConfig, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(cfg.getAPIURL("/api/v1/wallets/00000000-0000-0000-0000-000000000000"))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode != 500 {
				t.Logf("API is ready at %s", cfg.BaseURL)
				return
			}
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatalf("API not ready after %v", timeout)
}

func createWalletHelper(t *testing.T, cfg *TestConfig) string {
	walletID := uuid.New().String()

	reqBody := map[string]interface{}{
		"walletId":      walletID,
		"operationType": "DEPOSIT",
		"amount":        100,
	}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := http.Post(cfg.getAPIURL("/api/v1/wallet"), "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to create wallet, status: %d", resp.StatusCode)
	}

	return walletID
}

func TestConcurrency_1000Operations_SameWallet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cfg := LoadTestConfig(t)

	waitForAPI(t, cfg, 30*time.Second)

	walletID := createWalletHelper(t, cfg)

	t.Logf("Testing with API URL: %s", cfg.BaseURL)
	t.Logf("Wallet ID: %s", walletID)

	var wg sync.WaitGroup
	operations := 1000
	amountPerOp := int64(10)

	var successCount atomic.Int32
	var error4xxCount atomic.Int32
	var error5xxCount atomic.Int32

	startTime := time.Now()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for i := 0; i < operations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			reqBody := map[string]interface{}{
				"walletId":      walletID,
				"operationType": "DEPOSIT",
				"amount":        amountPerOp,
			}
			jsonBody, _ := json.Marshal(reqBody)

			resp, err := client.Post(cfg.getAPIURL("/api/v1/wallet"), "application/json", bytes.NewBuffer(jsonBody))
			if err != nil {
				t.Logf("Request failed: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 500 {
				error5xxCount.Add(1)
				t.Logf("Got 5xx error: %d", resp.StatusCode)
			} else if resp.StatusCode >= 400 {
				error4xxCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	t.Logf("Completed %d operations in %s", operations, elapsed)
	t.Logf("Success: %d, 4XX Errors: %d, 5XX Errors: %d",
		successCount.Load(), error4xxCount.Load(), error5xxCount.Load())

	if error5xxCount.Load() > 0 {
		t.Errorf("❌ FAILED: Got %d 5XX errors (violates requirement: no 50X errors)", error5xxCount.Load())
	} else {
		t.Logf("✅ PASSED: No 5XX errors")
	}

	time.Sleep(2 * time.Second)

	resp, err := client.Get(cfg.getAPIURL(fmt.Sprintf("/api/v1/wallets/%s", walletID)))
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]int64
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	expectedBalance := amountPerOp * int64(operations)
	if result["balance"] != expectedBalance {
		t.Errorf("Balance mismatch! Expected %d, got %d", expectedBalance, result["balance"])
	} else {
		t.Logf("✅ Balance verified: %d", result["balance"])
	}
}

func TestConcurrency_MixedOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cfg := LoadTestConfig(t)
	waitForAPI(t, cfg, 30*time.Second)

	walletID := createWalletHelper(t, cfg)

	t.Logf("Testing mixed operations with wallet: %s", walletID)

	var wg sync.WaitGroup
	operations := 500

	initialDeposit := int64(100000)
	reqBody := map[string]interface{}{
		"walletId":      walletID,
		"operationType": "DEPOSIT",
		"amount":        initialDeposit,
	}
	jsonBody, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(cfg.getAPIURL("/api/v1/wallet"), "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to make initial deposit: %v", err)
	}
	resp.Body.Close()

	var error5xxCount atomic.Int32

	expectedBalance := initialDeposit + int64(25000)

	for i := 0; i < operations; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()
			reqBody := map[string]interface{}{
				"walletId":      walletID,
				"operationType": "DEPOSIT",
				"amount":        100,
			}
			jsonBody, _ := json.Marshal(reqBody)
			resp, err := client.Post(cfg.getAPIURL("/api/v1/wallet"), "application/json", bytes.NewBuffer(jsonBody))
			if err == nil && resp.StatusCode >= 500 {
				error5xxCount.Add(1)
			}
			if resp != nil {
				resp.Body.Close()
			}
		}()

		go func() {
			defer wg.Done()
			reqBody := map[string]interface{}{
				"walletId":      walletID,
				"operationType": "WITHDRAW",
				"amount":        50,
			}
			jsonBody, _ := json.Marshal(reqBody)
			resp, err := client.Post(cfg.getAPIURL("/api/v1/wallet"), "application/json", bytes.NewBuffer(jsonBody))
			if err == nil && resp.StatusCode >= 500 {
				error5xxCount.Add(1)
			}
			if resp != nil {
				resp.Body.Close()
			}
		}()
	}

	wg.Wait()
	time.Sleep(2 * time.Second)

	resp, err = client.Get(cfg.getAPIURL(fmt.Sprintf("/api/v1/wallets/%s", walletID)))
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]int64
	json.NewDecoder(resp.Body).Decode(&result)

	if error5xxCount.Load() > 0 {
		t.Errorf("❌ Got %d 5XX errors", error5xxCount.Load())
	} else {
		t.Logf("✅ No 5XX errors in mixed operations")
	}

	if result["balance"] != expectedBalance {
		t.Errorf("Balance mismatch! Expected %d, got %d", expectedBalance, result["balance"])
	} else {
		t.Logf("✅ Mixed operations balance verified: %d", result["balance"])
	}
}

func TestConcurrency_1000RPS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	cfg := LoadTestConfig(t)
	waitForAPI(t, cfg, 30*time.Second)

	walletID := createWalletHelper(t, cfg)

	t.Logf("Starting stress test with 1000 RPS on wallet: %s", walletID)

	duration := 10 * time.Second
	requestsPerSecond := 1000

	var totalRequests atomic.Int64
	var error5xx atomic.Int64

	stopCh := make(chan struct{})

	numWorkers := 10
	var wg sync.WaitGroup

	requestsPerWorker := requestsPerSecond / numWorkers
	if requestsPerWorker == 0 {
		requestsPerWorker = 1
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ticker := time.NewTicker(time.Second / time.Duration(requestsPerWorker))
			defer ticker.Stop()

			for {
				select {
				case <-stopCh:
					return
				case <-ticker.C:
					totalRequests.Add(1)
					reqBody := map[string]interface{}{
						"walletId":      walletID,
						"operationType": "DEPOSIT",
						"amount":        1,
					}
					jsonBody, _ := json.Marshal(reqBody)
					resp, err := client.Post(cfg.getAPIURL("/api/v1/wallet"), "application/json", bytes.NewBuffer(jsonBody))
					if err == nil && resp.StatusCode >= 500 {
						error5xx.Add(1)
					}
					if resp != nil {
						resp.Body.Close()
					}
				}
			}
		}(w)
	}

	time.Sleep(duration)
	close(stopCh)
	wg.Wait()

	t.Logf("Stress test completed:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Target RPS: %d", requestsPerSecond)
	t.Logf("  Total requests: %d", totalRequests.Load())
	t.Logf("  Actual RPS: %.2f", float64(totalRequests.Load())/duration.Seconds())
	t.Logf("  5XX errors: %d", error5xx.Load())

	if error5xx.Load() > 0 {
		t.Errorf("❌ FAILED: Got %d 5XX errors (violates requirement: no 50X errors)", error5xx.Load())
		errorPercentage := float64(error5xx.Load()) / float64(totalRequests.Load()) * 100
		t.Errorf("5XX error rate: %.2f%%", errorPercentage)
	} else {
		t.Logf("✅ PASSED: No 5XX errors in stress test")
	}
}

func TestAPIHealthCheck(t *testing.T) {
	cfg := LoadTestConfig(t)

	waitForAPI(t, cfg, 10*time.Second)

	resp, err := http.Get(cfg.getAPIURL("/api/v1/wallets/00000000-0000-0000-0000-000000000000"))
	if err != nil {
		t.Skipf("API not available at %s: %v", cfg.BaseURL, err)
	}
	defer resp.Body.Close()

	t.Logf("✅ API is available at %s", cfg.BaseURL)
}
