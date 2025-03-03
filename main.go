package main

import (
	"fmt"
	"maps"
	"math/rand"
	"sync"
)


type Account struct {
	ID      string
	Balance int
}

type StateTransitions interface {
	Deposit(accountId string, amount int) error
	Withdraw(accountId string, amount int) error
	Transfer(fromAccountId, toAccountId string, amount int) error
	Rollback() error
}

type StateMachine struct {
	accounts map[string]int   // store current state => current balance of each account
	history  []map[string]int // => stores past states for rollback
	mu       sync.Mutex
}

func (sm *StateMachine) Deposit(accountId string, amount int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	fmt.Printf("\n\nDepositing %d to account %s\n", amount, accountId)

	sm.saveState()

	if _, ok := sm.accounts[accountId]; !ok {
		return fmt.Errorf("invalid account (%s) to deposit to", accountId)
	}

	sm.accounts[accountId] += amount

	fmt.Println("After Deposit:", sm.accounts)

	return nil
}

func (sm *StateMachine) Withdraw(accountId string, amount int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	fmt.Printf("\n\nWithdrawing %d from account %s\n", amount, accountId)

	sm.saveState()

	if _, ok := sm.accounts[accountId]; !ok {
		return fmt.Errorf("invalid account (%s) to withdraw from", accountId)
	}

	currentBalance := sm.accounts[accountId]
	if currentBalance < amount {
		return fmt.Errorf("insufficient balance (%d)", currentBalance)
	}

	sm.accounts[accountId] -= amount

	fmt.Println("After Withdraw:", sm.accounts)

	return nil
}

func (sm *StateMachine) Transfer(fromAccountId, toAccountId string, amount int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	fmt.Printf("\n\nTransfering %d from account %s to account %s\n", amount, fromAccountId, toAccountId)

	sm.saveState()

	if _, ok := sm.accounts[fromAccountId]; !ok {
		return fmt.Errorf("invalid sender account %s", fromAccountId)
	}

	if _, ok := sm.accounts[toAccountId]; !ok {
		return fmt.Errorf("invalid receiver account %s", toAccountId)
	}

	currentBalanceOfSender := sm.accounts[fromAccountId]
	if currentBalanceOfSender < amount {
		return fmt.Errorf("insufficient balance (%d) to transfer (%d) from", currentBalanceOfSender, amount)
	}

	sm.accounts[fromAccountId] -= amount
	sm.accounts[toAccountId] += amount

	fmt.Println("After transfer:", sm.accounts)

	return nil
}

func (sm *StateMachine) saveState() {
	snapshot := make(map[string]int)
	maps.Copy(snapshot, sm.accounts)
	sm.history = append(sm.history, snapshot)
}

func (sm *StateMachine) Rollback() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	historyLength := len(sm.history)
	if historyLength == 0 {
		return fmt.Errorf("nothing to rollback")
	}

	lastState := sm.history[historyLength-1]
	sm.accounts = lastState                   // reverse to the last state
	sm.history = sm.history[:historyLength-1] // delete the last state from history

	fmt.Println("After Rollback:", sm.accounts)

	return nil
}

func main() {
	var wg sync.WaitGroup
	noOfWorkers := 4

	sm := &StateMachine{
		accounts: map[string]int{
			"acc1": 1000,
			"acc2": 500,
			"acc3": 300,
		},
		history: []map[string]int{},
	}

	accountIds := []string{"acc1", "acc2", "acc3"}

	fmt.Println("Initial State:", sm.accounts)

	wg.Add(noOfWorkers)
	for range noOfWorkers {
		accountID := accountIds[rand.Intn(len(accountIds))]
		go func(id string) {
			defer wg.Done()
			if err := sm.Deposit(id, 200); err != nil {
				fmt.Println("Error:", err)
			} 
		}(accountID)
	}

	wg.Add(noOfWorkers)
	for range noOfWorkers {
		accountID := accountIds[rand.Intn(len(accountIds))]
		go func(id string) {
			defer wg.Done()
			if err := sm.Withdraw(id, 100); err != nil {
				fmt.Println("Error:", err)
			} 
		}(accountID)
	}

	wg.Add(noOfWorkers)
	for range noOfWorkers {
        fromAccountID := accountIds[rand.Intn(len(accountIds))]
        toAccountID := accountIds[rand.Intn(len(accountIds))]
        if fromAccountID != toAccountID {
            go func(fromID, toID string) {
                defer wg.Done()
                if err := sm.Transfer(fromID, toID, 75); err != nil {
                    fmt.Println("Transfer Error:", err)
                }
            }(fromAccountID, toAccountID)
        } else {
            wg.Done() // Avoid hanging if the same ID is chosen
        }
    }

	wg.Wait()

	fmt.Println("\nRolling back the last operation...")
	if err := sm.Rollback(); err != nil {
		fmt.Println("Error:", err)
	}

	if err := sm.Withdraw(accountIds[rand.Intn(len(accountIds))], 10000); err != nil {
		fmt.Println("Withdraw Error:", err)
	}

	fmt.Println("\nFinal State:", sm.accounts)
}
