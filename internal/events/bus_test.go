package events

import (
	"sync"
	"testing"
	"time"
)

func TestEmitAndReceive(t *testing.T) {
	bus := NewBus()
	ch, unsub := bus.Subscribe("/project")
	defer unsub()

	bus.Emit(Event{Type: TicketCreated, ProjectPath: "/project", TicketID: "t1"})

	select {
	case e := <-ch:
		if e.Type != TicketCreated {
			t.Errorf("type = %q, want %q", e.Type, TicketCreated)
		}
		if e.TicketID != "t1" {
			t.Errorf("ticketID = %q, want %q", e.TicketID, "t1")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewBus()
	ch1, unsub1 := bus.Subscribe("/project")
	defer unsub1()
	ch2, unsub2 := bus.Subscribe("/project")
	defer unsub2()

	bus.Emit(Event{Type: TicketUpdated, ProjectPath: "/project", TicketID: "t1"})

	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case e := <-ch:
			if e.Type != TicketUpdated {
				t.Errorf("subscriber %d: type = %q, want %q", i, e.Type, TicketUpdated)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out", i)
		}
	}
}

func TestProjectIsolation(t *testing.T) {
	bus := NewBus()
	chA, unsubA := bus.Subscribe("/projectA")
	defer unsubA()
	chB, unsubB := bus.Subscribe("/projectB")
	defer unsubB()

	bus.Emit(Event{Type: TicketCreated, ProjectPath: "/projectA", TicketID: "t1"})

	select {
	case <-chA:
		// expected
	case <-time.After(time.Second):
		t.Fatal("projectA subscriber should receive event")
	}

	select {
	case <-chB:
		t.Fatal("projectB subscriber should not receive projectA event")
	case <-time.After(50 * time.Millisecond):
		// expected
	}
}

func TestSlowConsumerNonBlocking(t *testing.T) {
	bus := NewBus()
	ch, unsub := bus.Subscribe("/project")
	defer unsub()

	// Fill the buffer (capacity 64)
	for range 100 {
		bus.Emit(Event{Type: TicketCreated, ProjectPath: "/project", TicketID: "t1"})
	}

	// Emit should not block, verify we can drain what's in the buffer
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != 64 {
		t.Errorf("expected 64 buffered events, got %d", count)
	}
}

func TestUnsubscribeClosesChannel(t *testing.T) {
	bus := NewBus()
	ch, unsub := bus.Subscribe("/project")
	unsub()

	_, ok := <-ch
	if ok {
		t.Error("channel should be closed after unsubscribe")
	}
}

func TestConcurrentSafety(t *testing.T) {
	bus := NewBus()

	var wg sync.WaitGroup
	wg.Add(3)

	// Concurrent subscriber
	go func() {
		defer wg.Done()
		for range 100 {
			_, unsub := bus.Subscribe("/project")
			unsub()
		}
	}()

	// Concurrent emitter
	go func() {
		defer wg.Done()
		for range 100 {
			bus.Emit(Event{Type: TicketCreated, ProjectPath: "/project", TicketID: "t1"})
		}
	}()

	// Concurrent subscribe+read
	go func() {
		defer wg.Done()
		ch, unsub := bus.Subscribe("/project")
		defer unsub()
		for range 50 {
			bus.Emit(Event{Type: TicketUpdated, ProjectPath: "/project", TicketID: "t2"})
		}
		// Drain
		for {
			select {
			case <-ch:
			default:
				return
			}
		}
	}()

	wg.Wait()
}
