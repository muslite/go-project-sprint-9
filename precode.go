package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммы
// сгенерированных чисел.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	// 1. Функция Generator
	// ...
	defer close(ch)
	var i int64 = 1
	for {
		select {
		case <-ctx.Done():
			return
		default:
			ch <- i
			fn(i)
			i++
		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64) {
	// 2. Функция Worker
	// ...
	/* 	for {
	v, ok := <-in
	if !ok {
		close(out)
		break
	} */
	// Предпочтительно чтение из канала вместо бесконечного цикла.
	defer close(out)
	for v := range in {
		out <- v
		time.Sleep(time.Millisecond)
	}
}

func main() {
	chIn := make(chan int64)

	// 3. Создание контекста
	// ...
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// для проверки будем считать количество и сумму отправленных чисел
	var inputSum int64   // сумма сгенерированных чисел
	var inputCount int64 // количество сгенерированных чисел

	// генерируем числа, считая параллельно их количество и сумму
	go Generator(ctx, chIn, func(i int64) {
		// Если предположить что геренатор распараллелят, то и в функции требуется проводить атомарные операции
		atomic.AddInt64(&inputSum, i)   // вместо inputSum += i
		atomic.AddInt64(&inputCount, 1) //inputCount++
	})

	const NumOut = 5 // количество обрабатывающих горутин и каналов
	// outs — слайс каналов, куда будут записываться числа из chIn
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		// создаём каналы и для каждого из них вызываем горутину Worker
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	// amounts — слайс, в который собирается статистика по горутинам
	amounts := make([]int64, NumOut)
	// chOut — канал, в который будут отправляться числа из горутин `outs[i]`
	chOut := make(chan int64, NumOut)

	var wg sync.WaitGroup

	// 4. Собираем числа из каналов outs
	// ...
	// "синхронизируем", дожидаемся завершения работы сборщиков.
	for i := 0; i < NumOut; i++ {
		wg.Add(1)
		go func(in_cn <-chan int64, cn int) {
			defer wg.Done()
			/* 			for {
			v, ok := <-in_cn
			if !ok {
				break
			} */
			// Предпочтительно чтение из канала вместо бесконечного цикла.
			for v := range in_cn {
				amounts[cn]++
				chOut <- v
			}
		}(outs[i], int(i))
	}

	go func() {
		// ждём завершения работы всех горутин для outs
		wg.Wait()
		// закрываем результирующий канал
		close(chOut)
	}()

	var count int64 // количество чисел результирующего канала
	var sum int64   // сумма чисел результирующего канала

	// 5. Читаем числа из результирующего канала
	// ...
	// Предпочтительно чтение из канала вместо бесконечного цикла.
	/* 	for {
	v, ok := <-chOut
	if !ok {
		break
	} */
	for v := range chOut {
		count++
		sum += v
	}

	fmt.Println("Количество чисел", inputCount, count)
	fmt.Println("Сумма чисел", inputSum, sum)
	fmt.Println("Разбивка по каналам", amounts)

	// проверка результатов
	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
	for _, v := range amounts {
		inputCount -= v
	}
	if inputCount != 0 {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}
}
