package notificacion

import (
	"encoding/json"
	"log/slog"
	"sync"
)

type Evento struct {
	Nombre string
	Data   string
}

type Broker struct {
	mutex        sync.RWMutex
	suscriptores map[string]map[chan Evento]bool
}

var Instancia = NuevoBroker()

func NuevoBroker() *Broker {
	return &Broker{
		suscriptores: make(map[string]map[chan Evento]bool),
	}
}

func (b *Broker) Suscribir(canal string) (chan Evento, func()) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	ch := make(chan Evento, 128)
	if _, ok := b.suscriptores[canal]; !ok {
		b.suscriptores[canal] = make(map[chan Evento]bool)
	}
	b.suscriptores[canal][ch] = true

	var once sync.Once
	desuscribir := func() {
		once.Do(func() {
			b.mutex.Lock()
			defer b.mutex.Unlock()
			if subs, ok := b.suscriptores[canal]; ok {
				if _, existe := subs[ch]; existe {
					delete(subs, ch)
					close(ch)
				}
				if len(subs) == 0 {
					delete(b.suscriptores, canal)
				}
			}
		})
	}

	return ch, desuscribir
}

func (b *Broker) Estadisticas() (canales int, suscriptores int) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	canales = len(b.suscriptores)
	for _, subs := range b.suscriptores {
		suscriptores += len(subs)
	}
	return canales, suscriptores
}

func (b *Broker) Publicar(canal string, nombreEvento string, data any) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	clientes := b.suscriptores[canal]
	if len(clientes) == 0 {
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Error("error serializando datos del evento", "canal", canal, "evento", nombreEvento, "err", err)
		return
	}

	ev := Evento{
		Nombre: nombreEvento,
		Data:   string(jsonData),
	}

	for ch := range clientes {
		select {
		case ch <- ev:
		default:
			slog.Warn("canal de notificacion lleno, mensaje omitido", "canal", canal)
		}
	}
}
