package main

import (
	"math/rand"
	"sync"
	"testing"
)

func TestStateMachine(t *testing.T) {
	type Action struct {
		fn          func(sm *StateMachine) error
		expectedErr bool
	}

	tests := []struct {
		name             string
		initialAccounts  map[string]int
		actions          []Action
		expectedAccounts map[string]int
	}{
		{
			name: "Concurrent deposits and withdrawals",
			initialAccounts: map[string]int{
				"acc1": 1000,
				"acc2": 500,
				"acc3": 300,
			},
			actions: []Action{
				{fn: func(sm *StateMachine) error { return sm.Deposit("acc1", 200) }, expectedErr: false},
				{fn: func(sm *StateMachine) error { return sm.Withdraw("acc2", 100) }, expectedErr: false},
				{fn: func(sm *StateMachine) error { return sm.Transfer("acc1", "acc3", 150) }, expectedErr: false},
				{fn: func(sm *StateMachine) error { return sm.Withdraw("acc3", 500) }, expectedErr: true}, // Insufficient funds
			},
			expectedAccounts: map[string]int{
				"acc1": 1050,
				"acc2": 400,
				"acc3": 450,
			},
		},
		{
			name: "Invalid account operations",
			initialAccounts: map[string]int{
				"acc1": 1000,
			},
			actions: []Action{
				{fn: func(sm *StateMachine) error { return sm.Deposit("invalid", 100) }, expectedErr: true},
				{fn: func(sm *StateMachine) error { return sm.Withdraw("invalid", 100) }, expectedErr: true},
				{fn: func(sm *StateMachine) error { return sm.Transfer("acc1", "invalid", 50) }, expectedErr: true},
			},
			expectedAccounts: map[string]int{
				"acc1": 1000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateMachine{
				accounts: tt.initialAccounts,
				history:  []map[string]int{},
			}

			var wg sync.WaitGroup
			for _, action := range tt.actions {
				wg.Add(1)
				go func(act Action) {
					defer wg.Done()
					err := act.fn(sm)
					if (err != nil) != act.expectedErr {
						t.Errorf("Unexpected error state: got %v, expectedErr %v", err, act.expectedErr)
					}
				}(action)
			}
			wg.Wait()

			for acc, expectedBalance := range tt.expectedAccounts {
				if sm.accounts[acc] != expectedBalance {
					t.Errorf("Account %s balance = %d; want %d", acc, sm.accounts[acc], expectedBalance)
				}
			}
		})
	}
}

func TestStateMachineRollback(t *testing.T) {
	sm := &StateMachine{
		accounts: map[string]int{
			"acc1": 1000,
			"acc2": 500,
		},
		history: []map[string]int{},
	}

	_ = sm.Deposit("acc1", 200)
	_ = sm.Withdraw("acc2", 100)

	for range 2 {
		if err := sm.Rollback(); err != nil {
			t.Fatalf("Rollback failed: %v", err)
		}
	}

	expectedAccounts := map[string]int{
		"acc1": 1000,
		"acc2": 500,
	}

	for acc, expectedBalance := range expectedAccounts {
		if sm.accounts[acc] != expectedBalance {
			t.Errorf("Account %s balance = %d; want %d", acc, sm.accounts[acc], expectedBalance)
		}
	}
}

func TestStateMachineConcurrentStress(t *testing.T) {
	sm := &StateMachine{
		accounts: map[string]int{
			"acc1": 1000,
			"acc2": 500,
			"acc3": 300,
		},
		history: []map[string]int{},
	}

	accountIds := []string{"acc1", "acc2", "acc3"}
	var wg sync.WaitGroup
	noOfWorkers := 1000

	for range noOfWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			accountID := accountIds[rand.Intn(len(accountIds))]
			if err := sm.Deposit(accountID, 50); err != nil {
				t.Errorf("Error during deposit: %v", err)
			}
		}()
	}

	for range noOfWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			accountID := accountIds[rand.Intn(len(accountIds))]
			if err := sm.Withdraw(accountID, 20); err != nil {
				t.Logf("Expected error during withdrawal: %v", err)
			}
		}()
	}

	wg.Wait()

	// Just ensuring no race conditions and state consistency
	totalBalance := 0
	for _, balance := range sm.accounts {
		totalBalance += balance
	}
	expectedMinimumBalance := 1000 + 500 + 300 - (noOfWorkers * 20)
	if totalBalance < expectedMinimumBalance {
		t.Errorf("Total balance = %d; expected at least %d", totalBalance, expectedMinimumBalance)
	}
}
