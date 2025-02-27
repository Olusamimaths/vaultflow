package main

import (
	"fmt"
	"maps"
)

type TransactionType = string

type Account struct {
	ID      string
	Balance int
}

type Transaction struct {
	Type   TransactionType
	Amount int
	From   string
	To     string
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
}

func (sm *StateMachine) Deposit(accountId string, amount int) error {
	sm.saveState()

	if _, ok := sm.accounts[accountId]; !ok {
		return fmt.Errorf("invalid account (%s) to deposit to", accountId)
	}

	sm.accounts[accountId] += amount

	return nil
}

func (sm *StateMachine) Withdraw(accountId string, amount int) error {
	sm.saveState()

	if _, ok := sm.accounts[accountId]; !ok {
		return fmt.Errorf("invalid account (%s) to withdraw from", accountId)
	}

	currentBalance := sm.accounts[accountId]
	if currentBalance < amount {
		return fmt.Errorf("insufficient balance (%d)", currentBalance)
	}

	sm.accounts[accountId] -= amount

	return nil
}

func (sm *StateMachine) Transfer(fromAccountId, toAccountId string, amount int) error {
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

	return nil
}

func (sm *StateMachine) saveState() {
	snapshot := make(map[string]int)
	maps.Copy(snapshot, sm.accounts)
	sm.history = append(sm.history, snapshot)
}

func (sm *StateMachine) Rollback() error {
	historyLength := len(sm.history)
	if historyLength == 0 {
		return fmt.Errorf("nothing to rollback")
	}

	lastState := sm.history[historyLength-1]
	sm.accounts = lastState // reverse to the last state
	sm.history = sm.history[:historyLength-1] // delete the last state from history

	return nil
}