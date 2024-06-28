package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// GameRound представляє ігровий раунд з питанням та варіантами відповідей
type GameRound struct {
	Question string
	Options  []string
}

// PlayerResponse представляє відповідь гравця
type PlayerResponse struct {
	PlayerID int
	Answer   int
}

// gameGenerator генерує новий ігровий раунд кожні 10 секунд
func gameGenerator(ctx context.Context, gameChan chan<- GameRound) {
	questions := []string{
		"What is the capital of France?",
		"What is 2 + 2?",
		"What is the capital of Spain?",
	}
	options := [][]string{
		{"Paris", "London", "Rome", "Berlin"},
		{"3", "4", "5", "6"},
		{"Madrid", "Lisbon", "Barcelona", "Seville"},
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			i := rand.Intn(len(questions))
			gameChan <- GameRound{Question: questions[i], Options: options[i]}
		}
	}
}

// player отримує новий ігровий раунд та відправляє свою відповідь
func player(ctx context.Context, playerID int, gameChan <-chan GameRound, responseChan chan<- PlayerResponse) {
	for {
		select {
		case <-ctx.Done():
			return
		case round := <-gameChan:
			fmt.Printf("Player %d received question: %s\n", playerID, round.Question)
			answer := rand.Intn(len(round.Options)) // Random answer
			responseChan <- PlayerResponse{PlayerID: playerID, Answer: answer}
		}
	}
}

// resultCounter підраховує відповіді гравців та відправляє результати у головну горутину
func resultCounter(ctx context.Context, responseChan <-chan PlayerResponse, resultChan chan<- map[int]int) {
	results := make(map[int]int)

	for {
		select {
		case <-ctx.Done():
			return
		case response := <-responseChan:
			results[response.Answer]++
			resultCopy := make(map[int]int)
			for k, v := range results {
				resultCopy[k] = v
			}
			resultChan <- resultCopy
		}
	}
}

func main() {
	gameChan := make(chan GameRound)
	responseChan := make(chan PlayerResponse)
	resultChan := make(chan map[int]int)

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	// Запускаємо генератор ігрових раундів
	wg.Add(1)
	go func() {
		defer wg.Done()
		gameGenerator(ctx, gameChan)
	}()

	// Запускаємо гравців
	numPlayers := 5
	playerChans := make([]chan GameRound, numPlayers)
	for i := 0; i < numPlayers; i++ {
		playerChans[i] = make(chan GameRound)
		wg.Add(1)
		go func(playerID int, playerChan <-chan GameRound) {
			defer wg.Done()
			player(ctx, playerID, playerChan, responseChan)
		}(i+1, playerChans[i])
	}

	// Фан-аут горутина для розподілу ігрових раундів всім гравцям
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case round := <-gameChan:
				for _, ch := range playerChans {
					ch <- round
				}
			}
		}
	}()

	// Запускаємо лічильник результатів
	wg.Add(1)
	go func() {
		defer wg.Done()
		resultCounter(ctx, responseChan, resultChan)
	}()

	// Виводимо результати
	go func() {
		for result := range resultChan {
			fmt.Printf("Current results: %v\n", result)
		}
	}()

	// Чекаємо на завершення програми користувачем
	fmt.Println("Press Enter to stop the game...")
	fmt.Scanln()
	cancel()  // Завершуємо всі горутини через контекст
	wg.Wait() // Чекаємо на завершення всіх горутин

	fmt.Println("Game stopped.")
}
