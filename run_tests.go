package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// Lista de partidas a serem testadas.
var gameMoves = []string{
	//"e2e4 e7e5 g1f3 b8c6 f1c4 d8f6 e1g1", // Roque
	//"e2e4 e7e5 g1f3 f7f5 d2d4 f5e4 f3e5 e4d3", // en passant
	"d2d4 b7b5 d4d5 b5b4 d5d6 b4b3 d6c7 b3c2 c7b8q", // promotion //TODO: Want fix
	// "e2e4 e7e5 d1h5 b8c6 f1c4 g8f6 h5f7", // white win
	// "c2c4 h7h5 h2h4 a7a5 d1a4 a8a6 a4a5 a6h6 a5c7 f7f6 c7d7 e8f7 d7b7 d7b8 b7b8 h6h7 b8c8 d7d7" //stalemate
}

func main() {
	fmt.Println("Starting Go Chess Engine Test Runner...")

	for i, movesString := range gameMoves {
		fmt.Printf("--- Iniciando partida de teste #%d ---\n", i+1)

		// Roda o seu executável Go como um subprocesso.
		// As flags stdin, stdout e stderr são para capturar o I/O do processo.
		cmd := exec.Command("./bin/chess")
		stdin, err := cmd.StdinPipe()
		if err != nil {
			panic(err)
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		cmd.Stderr = cmd.Stdout

		if err := cmd.Start(); err != nil {
			panic(err)
		}

		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				fmt.Println(scanner.Text())
			}
		}()

		moves := strings.Split(movesString, " ")

		for _, move := range moves {
			fmt.Printf("Enviando jogada: %s\n", move)
			io.WriteString(stdin, move+"\n")
			time.Sleep(1 * time.Second)
		}

		fmt.Println("Enviando comando de saída: exit")
		io.WriteString(stdin, "exit\n")
		stdin.Close()

		cmd.Wait()

		fmt.Printf("--- Fim da partida de teste #%d ---\n\n", i+1)
		time.Sleep(2 * time.Second)
	}

	fmt.Println("Todos os testes concluídos.")
}
