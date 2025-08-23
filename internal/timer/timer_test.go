package timer

import (
	"sync"
	"testing"
	"time"

	common "idylicaro/go-chess/internal/common"
)

// Ajudante para verificar o tempo com uma pequena margem de erro
func checkRemainingTime(t *testing.T, timer *GameTimer, color common.Color, expected time.Duration) {
	t.Helper()
	// Use uma pequena margem de erro para levar em conta a precisão da máquina
	delta := 20 * time.Millisecond
	actual := timer.GetRemainingTime(color)
	if actual < expected-delta || actual > expected+delta {
		t.Errorf("Tempo restante para %s incorreto. Esperado: %v, Recebido: %v", color, expected, actual)
	}
}

// Teste o fluxo de jogo completo com incremento
func TestGameTimer_FullFlow(t *testing.T) {
	// Configurações do timer: 10 segundos iniciais, 1 segundo de incremento
	config := TimeConfig{InitialTime: 10 * time.Second, Increment: 1 * time.Second}
	gt := NewGameTimer(config)

	// Inicia o timer para as Brancas
	gt.Start(common.White)
	if !gt.white.active {
		t.Error("Relógio das Brancas não foi ativado corretamente")
	}

	// Simula 2 segundos de jogo para as Brancas
	time.Sleep(2 * time.Second)

	// Verifica o tempo restante das Brancas
	// Esperado: 10s (inicial) - 2s (decorrido) = 8s
	checkRemainingTime(t, gt, common.White, 8*time.Second)

	// Troca para as Pretas
	gt.SwitchPlayer(common.Black)
	if !gt.black.active {
		t.Error("Relógio das Pretas não foi ativado após a troca")
	}
	// Verifica o tempo restante das Brancas após o incremento
	// Esperado: 8s (anterior) + 1s (incremento) = 9s
	checkRemainingTime(t, gt, common.White, 9*time.Second)

	// Simula 3 segundos de jogo para as Pretas
	time.Sleep(3 * time.Second)

	// Verifica o tempo restante das Pretas
	// Esperado: 10s (inicial) - 3s (decorrido) = 7s
	checkRemainingTime(t, gt, common.Black, 7*time.Second)

	// Troca de volta para as Brancas
	gt.SwitchPlayer(common.White)
	// Verifica o tempo restante das Pretas após o incremento
	// Esperado: 7s (anterior) + 1s (incremento) = 8s
	checkRemainingTime(t, gt, common.Black, 8*time.Second)
}

// Teste a condição de Timeout
func TestGameTimer_Timeout(t *testing.T) {
	// Use uma pequena quantidade de tempo para facilitar o teste de timeout
	config := TimeConfig{InitialTime: 100 * time.Millisecond, Increment: 0}
	gt := NewGameTimer(config)

	timeoutColor := common.White
	var wg sync.WaitGroup
	wg.Add(1)

	// Define a função de callback
	gt.SetTimeoutCallback(func(color common.Color) {
		timeoutColor = color
		wg.Done()
	})

	// Inicia e simula um timeout
	gt.Start(common.White)
	time.Sleep(200 * time.Millisecond) // Dorme por mais tempo do que o tempo inicial
	gt.SwitchPlayer(common.Black)      // Dispara o timeout

	// Espera pelo callback
	wg.Wait()

	if timeoutColor != common.White {
		t.Errorf("Timeout incorreto. Esperado: %s, Recebido: %s", common.White, timeoutColor)
	}

	// A função IsTimeout deve retornar verdadeiro para as Brancas
	if !gt.IsTimeout(common.White) {
		t.Error("IsTimeout deveria ser true para as Brancas")
	}
}

// Teste a função GetRemainingTime para quando o timer está inativo
func TestGameTimer_GetRemainingTime_Inactive(t *testing.T) {
	config := TimeConfig{InitialTime: 10 * time.Second}
	gt := NewGameTimer(config)

	// O timer está inativo inicialmente.
	remaining := gt.GetRemainingTime(common.White)
	if remaining != 10*time.Second {
		t.Errorf("Tempo restante incorreto para jogador inativo. Esperado: %v, Recebido: %v", 10*time.Second, remaining)
	}

	// Inicia o timer, simula um pouco de tempo e depois o desativa.
	gt.Start(common.White)
	time.Sleep(1 * time.Second)
	gt.SwitchPlayer(common.Black)

	// O tempo das Brancas deve ter o tempo correto, mesmo estando inativo
	checkRemainingTime(t, gt, common.White, 9*time.Second)
}

// Teste para garantir que a função SetTimeoutCallback funciona
func TestGameTimer_SetTimeoutCallback(t *testing.T) {
	config := TimeConfig{InitialTime: 1 * time.Second}
	gt := NewGameTimer(config)
	var callbackCalled bool
	gt.SetTimeoutCallback(func(c common.Color) {
		callbackCalled = true
	})

	// Simula um timeout para acionar a função de callback
	gt.Start(common.White)
	time.Sleep(2 * time.Second)
	gt.SwitchPlayer(common.Black)

	if !callbackCalled {
		t.Error("A função de callback não foi chamada")
	}
}
