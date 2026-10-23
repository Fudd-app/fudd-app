package pedido

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/evento"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/notificacion"
)

type Service struct {
	store Store
}

func NuevoService(s Store) *Service { return &Service{store: s} }

func (svc *Service) Crear(ctx context.Context, input NuevoPedidoInput) (*Pedido, error) {
	input.Origen = "comensal"
	input.CreadoPorUsuarioID = ""
	input.CreadoPorNombre = ""
	ctx = evento.ContextoConActor(ctx, evento.Comensal(primerNombreComensal(input.Items)))
	return svc.crear(ctx, input)
}

func (svc *Service) CrearAsistido(ctx context.Context, actor *auth.Claims, input PedidoAsistidoInput) (*Pedido, error) {
	if actor == nil || (actor.Rol != auth.RolDuenio && actor.Rol != auth.RolEncargado && actor.Rol != auth.RolMozo) {
		return nil, errors.New("sin permisos para cargar pedidos")
	}
	sucursalID, _, err := svc.store.ObtenerSucursalPorMesa(ctx, input.MesaID)
	if err != nil {
		return nil, err
	}
	if actor.Rol != auth.RolDuenio && sucursalID != actor.SucursalID {
		return nil, errors.New("mesa fuera de la sucursal asignada")
	}
	comensalID, err := nuevoUUID()
	if err != nil {
		return nil, err
	}
	alias := strings.TrimSpace(input.AliasCliente)
	for i := range input.Items {
		if input.Items[i].ComensalID == "" {
			input.Items[i].ComensalID = comensalID
		}
		if input.Items[i].ComensalNombre == "" {
			input.Items[i].ComensalNombre = alias
		}
	}
	ctx = evento.ContextoConActor(ctx, evento.Actor{Tipo: evento.ActorUsuario, UsuarioID: actor.UsuarioID, Rol: actor.Rol})
	return svc.crear(ctx, NuevoPedidoInput{
		MesaID: input.MesaID, Items: input.Items, Origen: "mozo",
		CreadoPorUsuarioID: actor.UsuarioID, CreadoPorNombre: aliasCreador(actor),
		// Lo carga personal que podría confirmarlo, así que nace confirmado.
		EstadoInicial: "confirmado",
	})
}

func primerNombreComensal(items []NuevoItemInput) string {
	for _, it := range items {
		if nombre := strings.TrimSpace(it.ComensalNombre); nombre != "" {
			return nombre
		}
	}
	return ""
}

func aliasCreador(actor *auth.Claims) string {
	// El nombre se completa en la persistencia histórica desde el usuario en
	// una mejora futura; el rol evita perder contexto si luego se lo elimina.
	return actor.Rol
}

func (svc *Service) crear(ctx context.Context, input NuevoPedidoInput) (*Pedido, error) {
	if input.MesaID == "" {
		return nil, fmt.Errorf("mesa_id requerido: %w", ErrValidation)
	}
	if len(input.Items) == 0 {
		return nil, fmt.Errorf("el pedido debe tener al menos un ítem: %w", ErrValidation)
	}
	for i := range input.Items {
		item := &input.Items[i]
		if item.Cantidad <= 0 {
			return nil, fmt.Errorf("cantidad inválida para artículo %s: %w", item.ArticuloID, ErrValidation)
		}
		item.ComensalNombre = strings.TrimSpace(item.ComensalNombre)
		if !uuidValido(item.ComensalID) {
			return nil, fmt.Errorf("comensal_id inválido para artículo %s: %w", item.ArticuloID, ErrValidation)
		}
		if utf8.RuneCountInString(item.ComensalNombre) > 100 {
			return nil, fmt.Errorf("comensal_nombre inválido para artículo %s: %w", item.ArticuloID, ErrValidation)
		}
		for _, vID := range item.Variantes {
			if strings.TrimSpace(vID) == "" {
				return nil, fmt.Errorf("variante_id no puede estar vacío: %w", ErrValidation)
			}
		}
	}
	sucursalID, cuentaVersion, err := svc.store.ObtenerSucursalPorMesa(ctx, input.MesaID)
	if err != nil {
		if errors.Is(err, ErrMesaCerrada) {
			return nil, fmt.Errorf("mesa cerrada: %w", ErrMesaCerrada)
		}
		if errors.Is(err, ErrCuentaSolicitada) {
			return nil, fmt.Errorf("la cuenta ya fue solicitada: %w", ErrCuentaSolicitada)
		}
		return nil, fmt.Errorf("mesa no encontrada: %w", ErrNotFound)
	}
	p, err := svc.store.Crear(ctx, input, sucursalID, cuentaVersion)
	if err != nil {
		if errors.Is(err, ErrCuentaSolicitada) {
			return nil, fmt.Errorf("la cuenta ya fue solicitada o cerrada: %w", ErrCuentaSolicitada)
		}
		return nil, err
	}

	// Notificar en tiempo real al recepcionista de la sucursal
	notificacion.Instancia.Publicar(fmt.Sprintf("sucursal:%s", p.SucursalID), "pedido_creado", p)
	notificacion.Instancia.Publicar(fmt.Sprintf("mesa:%s", p.MesaID), "pedido_creado", p)
	// Cocina solo ve pedidos confirmados; si nace "recibido" se entera por pedido_actualizado al confirmarse.
	if p.Estado == "confirmado" {
		notificacion.Instancia.Publicar(fmt.Sprintf("kds:%s", p.SucursalID), "pedido_creado", p)
	}

	return p, nil
}

func (svc *Service) ListarActivos(ctx context.Context, sucursalID, tenantID string) ([]Pedido, error) {
	return svc.store.ListarActivos(ctx, sucursalID, tenantID)
}

func (svc *Service) ListarCuentaActualPorQR(ctx context.Context, qrToken string) ([]Pedido, error) {
	if strings.TrimSpace(qrToken) == "" {
		return nil, fmt.Errorf("qr_token requerido: %w", ErrValidation)
	}
	return svc.store.ListarCuentaActualPorQR(ctx, qrToken)
}

// rangoEstado ordena el flujo del pedido; solo se puede avanzar.
var rangoEstado = map[string]int{"recibido": 0, "confirmado": 1, "preparando": 2, "listo": 3, "entregado": 4, "cerrado": 5}

// puedeLlegarA indica qué roles pueden llevar un pedido al estado destino.
// El mozo conserva preparando/listo para no romper a los clientes anteriores al flujo de confirmación.
func puedeLlegarA(rol, destino string) bool {
	switch destino {
	case "confirmado", "entregado", "cerrado":
		return rol == auth.RolDuenio || rol == auth.RolEncargado || rol == auth.RolMozo
	case "preparando", "listo":
		return rol == auth.RolDuenio || rol == auth.RolEncargado || rol == auth.RolMozo || rol == auth.RolCocinero
	}
	return false
}

// ValidarTransicion verifica que pasar de anterior a nuevo sea un avance válido del flujo.
// Repetir el mismo estado no es una transición y se acepta sin efecto.
func ValidarTransicion(anterior, nuevo string) error {
	if anterior == nuevo {
		return nil
	}
	if anterior == "cerrado" || nuevo == "recibido" {
		return fmt.Errorf("no se puede pasar de %q a %q: %w", anterior, nuevo, ErrTransicionInvalida)
	}
	if nuevo == "cerrado" {
		return nil
	}
	if rangoEstado[nuevo] <= rangoEstado[anterior] {
		return fmt.Errorf("no se puede pasar de %q a %q: %w", anterior, nuevo, ErrTransicionInvalida)
	}
	if nuevo == "entregado" && anterior != "listo" {
		return fmt.Errorf("solo un pedido listo puede pasar a entregado (estado actual %q): %w", anterior, ErrTransicionInvalida)
	}
	if anterior == "recibido" && (nuevo == "preparando" || nuevo == "listo") {
		return fmt.Errorf("el pedido debe confirmarse antes de pasar a %q: %w", nuevo, ErrPedidoSinConfirmar)
	}
	return nil
}

func (svc *Service) CambiarEstado(ctx context.Context, actor *auth.Claims, id string, input CambiarEstadoInput) (*Pedido, error) {
	nuevoEstado := input.Estado
	if !estadoValido(nuevoEstado) {
		return nil, fmt.Errorf("estado inválido: %q (válidos: %s): %w", nuevoEstado, strings.Join(EstadosValidos, ", "), ErrValidation)
	}
	modoEntrega := ""
	if nuevoEstado != "entregado" && strings.TrimSpace(input.ModoEntrega) != "" {
		return nil, fmt.Errorf("modo_entrega solo aplica al pasar a %q: %w", "entregado", ErrModoEntregaInvalido)
	}
	if nuevoEstado == "entregado" {
		modoEntrega = strings.TrimSpace(input.ModoEntrega)
		if modoEntrega != ModoEntregaMesa && modoEntrega != ModoEntregaRetiro {
			return nil, fmt.Errorf("modo_entrega es obligatorio y debe ser %q o %q: %w", ModoEntregaMesa, ModoEntregaRetiro, ErrModoEntregaInvalido)
		}
	}
	if nuevoEstado == "recibido" {
		return nil, fmt.Errorf("un pedido no puede volver a %q: %w", nuevoEstado, ErrTransicionInvalida)
	}
	if actor == nil || !puedeLlegarA(actor.Rol, nuevoEstado) {
		return nil, fmt.Errorf("el rol no puede pasar pedidos a %q: %w", nuevoEstado, ErrSinPermiso)
	}
	p, err := svc.store.CambiarEstado(ctx, id, actor.TenantID, actor.SucursalID, nuevoEstado, modoEntrega)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrTransicionInvalida) || errors.Is(err, ErrPedidoSinConfirmar) {
			return nil, err
		}
		return nil, fmt.Errorf("error cambiando estado: %w", err)
	}

	// Notificar en tiempo real al comensal (pedido), al recepcionista (sucursal) y a cocina (kds)
	notificacion.Instancia.Publicar(fmt.Sprintf("pedido:%s", p.ID), "pedido_actualizado", p)
	notificacion.Instancia.Publicar(fmt.Sprintf("sucursal:%s", p.SucursalID), "pedido_actualizado", p)
	notificacion.Instancia.Publicar(fmt.Sprintf("mesa:%s", p.MesaID), "pedido_actualizado", p)
	notificacion.Instancia.Publicar(fmt.Sprintf("kds:%s", p.SucursalID), "pedido_actualizado", p)

	return p, nil
}

func (svc *Service) CambiarEstadoItem(ctx context.Context, itemID, tenantID, sucursalAlcance, nuevoEstado string) (*Pedido, error) {
	itemID = strings.TrimSpace(itemID)
	nuevoEstado = strings.TrimSpace(nuevoEstado)
	if itemID == "" {
		return nil, fmt.Errorf("id de item requerido: %w", ErrValidation)
	}
	if !estadoItemValido(nuevoEstado) {
		return nil, fmt.Errorf("estado de item inválido: %q (válidos: pendiente, preparando, listo): %w", nuevoEstado, ErrValidation)
	}

	p, err := svc.store.CambiarEstadoItem(ctx, itemID, tenantID, sucursalAlcance, nuevoEstado)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrPedidoCerrado) ||
			errors.Is(err, ErrPedidoEntregado) || errors.Is(err, ErrPedidoSinConfirmar) {
			return nil, err
		}
		return nil, fmt.Errorf("error cambiando estado del item: %w", err)
	}

	// Cocina recibe un evento exclusivo y el resto de las vistas conserva
	// la actualización global del pedido para no desincronizar salón y mesa.
	notificacion.Instancia.Publicar(fmt.Sprintf("kds:%s", p.SucursalID), "pedido_item_actualizado", p)
	notificacion.Instancia.Publicar(fmt.Sprintf("pedido:%s", p.ID), "pedido_actualizado", p)
	notificacion.Instancia.Publicar(fmt.Sprintf("sucursal:%s", p.SucursalID), "pedido_actualizado", p)
	notificacion.Instancia.Publicar(fmt.Sprintf("mesa:%s", p.MesaID), "pedido_actualizado", p)

	return p, nil
}

func (svc *Service) ValidarAccesoKDS(ctx context.Context, sucursalID, tenantID string) error {
	sucursalID = strings.TrimSpace(sucursalID)
	if sucursalID == "" {
		return fmt.Errorf("sucursal_id requerido: %w", ErrValidation)
	}
	pertenece, err := svc.store.SucursalPerteneceATenant(ctx, sucursalID, tenantID)
	if err != nil {
		return fmt.Errorf("error validando acceso a cocina: %w", err)
	}
	if !pertenece {
		return ErrNotFound
	}
	return nil
}

func uuidValido(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	compacto := strings.ReplaceAll(value, "-", "")
	_, err := hex.DecodeString(compacto)
	return err == nil
}

func nuevoUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32], nil
}

func estadoValido(estado string) bool {
	for _, v := range EstadosValidos {
		if v == estado {
			return true
		}
	}
	return false
}

func estadoItemValido(estado string) bool {
	for _, valido := range EstadosItemValidos {
		if valido == estado {
			return true
		}
	}
	return false
}
