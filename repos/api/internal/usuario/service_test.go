package usuario_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/usuario"
)

type equipoStore struct {
	crearFn        func(string, string, string) (*usuario.Invitacion, error)
	preview        *usuario.Invitacion
	aceptada       bool
	nombreAceptado string
}

type emailSpy struct {
	email string
	link  string
	err   error
}

func (e *emailSpy) EnviarMagicLink(_ context.Context, email, link string) error {
	e.email = email
	e.link = link
	return e.err
}

func (m *equipoStore) SucursalPertenece(context.Context, string, string) (bool, error) {
	return true, nil
}
func (m *equipoStore) CrearInvitacion(_ context.Context, _, suc, email, rol, _, _ string, exp time.Time) (*usuario.Invitacion, error) {
	if m.crearFn != nil {
		return m.crearFn(suc, email, rol)
	}
	return &usuario.Invitacion{ID: "i-1", SucursalID: suc, Email: email, Rol: rol, Estado: "pendiente", ExpiresAt: exp}, nil
}
func (*equipoStore) ListarUsuarios(context.Context, string, string) ([]usuario.Usuario, error) {
	return []usuario.Usuario{}, nil
}
func (*equipoStore) ListarInvitaciones(context.Context, string, string) ([]usuario.Invitacion, error) {
	return []usuario.Invitacion{}, nil
}
func (*equipoStore) ObtenerUsuario(context.Context, string, string) (*usuario.Usuario, error) {
	return nil, usuario.ErrNotFound
}
func (*equipoStore) ActualizarUsuario(context.Context, string, string, string, string) (*usuario.Usuario, error) {
	return nil, nil
}
func (*equipoStore) EliminarUsuario(context.Context, string, string) error            { return nil }
func (*equipoStore) CancelarInvitacion(context.Context, string, string, string) error { return nil }
func (*equipoStore) ReenviarInvitacion(context.Context, string, string, string, string, string, time.Time) (*usuario.Invitacion, error) {
	return nil, nil
}
func (m *equipoStore) ObtenerInvitacionPorToken(context.Context, string) (*usuario.Invitacion, error) {
	if m.preview == nil {
		return nil, usuario.ErrInvitacionNoValida
	}
	return m.preview, nil
}
func (m *equipoStore) AceptarInvitacion(_ context.Context, _, _, nombre string, _ time.Time) (*usuario.Usuario, error) {
	m.aceptada = true
	m.nombreAceptado = nombre
	return &usuario.Usuario{ID: "u-1"}, nil
}

func TestEncargadoInvitaSoloRolesOperativosEnSuSucursal(t *testing.T) {
	var sucursal, rol string
	store := &equipoStore{crearFn: func(s, _ string, r string) (*usuario.Invitacion, error) {
		sucursal, rol = s, r
		return &usuario.Invitacion{SucursalID: s, Rol: r}, nil
	}}
	svc := usuario.NuevoService(store, nil, nil)
	actor := &auth.Claims{UsuarioID: "jefe", TenantID: "t-1", Rol: auth.RolEncargado, SucursalID: "s-propia"}
	if _, err := svc.Invitar(context.Background(), actor, usuario.InvitarInput{Email: "Mozo@Bar.com", Rol: auth.RolMozo, SucursalID: "s-ajena"}); err != nil {
		t.Fatal(err)
	}
	if sucursal != "s-propia" || rol != auth.RolMozo {
		t.Fatalf("alcance incorrecto: %s %s", sucursal, rol)
	}
	if _, err := svc.Invitar(context.Background(), actor, usuario.InvitarInput{Email: "jefe2@bar.com", Rol: auth.RolEncargado}); !errors.Is(err, usuario.ErrForbidden) {
		t.Fatalf("esperaba forbidden: %v", err)
	}
}

func TestInvitarEnviaEmailYDevuelveLinkParaCompartir(t *testing.T) {
	store := &equipoStore{}
	email := &emailSpy{}
	svc := usuario.NuevoService(store, nil, email)
	actor := &auth.Claims{UsuarioID: "duenio", TenantID: "t-1", Rol: auth.RolDuenio}

	respuesta, err := svc.Invitar(context.Background(), actor, usuario.InvitarInput{
		Email:      " Persona@Ejemplo.com ",
		Rol:        auth.RolMozo,
		SucursalID: "s-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if email.email != "persona@ejemplo.com" || email.link == "" {
		t.Fatalf("el email no recibió la invitación normalizada: email=%q link=%q", email.email, email.link)
	}
	if !respuesta.EmailEnviado {
		t.Fatal("la respuesta debe confirmar el envío al proveedor")
	}
	if respuesta.Link == "" || respuesta.Link != email.link {
		t.Fatal("el mismo enlace enviado debe devolverse para compartirlo manualmente")
	}
}

func TestInvitarSinProveedorRealNoInformaEnvioExitoso(t *testing.T) {
	store := &equipoStore{}
	svc := usuario.NuevoService(store, nil, &auth.LogEmailSender{})
	actor := &auth.Claims{UsuarioID: "duenio", TenantID: "t-1", Rol: auth.RolDuenio}

	respuesta, err := svc.Invitar(context.Background(), actor, usuario.InvitarInput{
		Email:      "persona@ejemplo.com",
		Rol:        auth.RolMozo,
		SucursalID: "s-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if respuesta.EmailEnviado {
		t.Fatal("el emisor de log no debe presentarse como un email enviado")
	}
	if respuesta.Link == "" || respuesta.AdvertenciaEmail == "" {
		t.Fatal("debe conservar el enlace y explicar el fallback manual")
	}
}

type authStore struct{}

func (*authStore) ObtenerUsuarioPorEmail(context.Context, string) (*auth.UsuarioAuth, error) {
	return nil, auth.ErrUsuarioNoEncontrado
}
func (*authStore) ObtenerUsuarioPorGoogleSub(context.Context, string) (*auth.UsuarioAuth, error) {
	return nil, auth.ErrUsuarioNoEncontrado
}
func (*authStore) ObtenerUsuarioPorID(context.Context, string) (*auth.UsuarioAuth, error) {
	return nil, auth.ErrUsuarioNoEncontrado
}
func (*authStore) VincularGoogleSub(context.Context, string, string) error { return nil }
func (*authStore) GuardarToken(context.Context, string, string, time.Time) (string, error) {
	return "", nil
}
func (*authStore) ObtenerToken(context.Context, string) (*auth.MagicToken, error) { return nil, nil }
func (*authStore) MarcarTokenUsado(context.Context, string) error                 { return nil }

type verifier struct{ email string }

func (v *verifier) Verificar(context.Context, string, string) (*auth.IdentidadGoogle, error) {
	return &auth.IdentidadGoogle{Subject: "g-1", Email: v.email, Nombre: "Persona", EmailVerificado: true}, nil
}

func TestAceptarRechazaOtraCuentaGoogleSinConsumir(t *testing.T) {
	store := &equipoStore{preview: &usuario.Invitacion{Email: "invitado@bar.com", Estado: "pendiente", ExpiresAt: time.Now().Add(time.Hour)}}
	authSvc := auth.NuevoServiceConGoogle(&authStore{}, nil, "client", &verifier{email: "otra@bar.com"})
	svc := usuario.NuevoService(store, authSvc, nil)
	_, err := svc.Aceptar(context.Background(), "token", usuario.AceptarInvitacionInput{CredencialGoogle: "cred"})
	if !errors.Is(err, usuario.ErrEmailGoogleDistinto) {
		t.Fatalf("esperaba email distinto: %v", err)
	}
}

func TestVistaPreviaExpirada(t *testing.T) {
	store := &equipoStore{preview: &usuario.Invitacion{Estado: "pendiente", ExpiresAt: time.Now().Add(-time.Minute)}}
	svc := usuario.NuevoService(store, nil, nil)
	_, err := svc.VistaPrevia(context.Background(), "token")
	if !errors.Is(err, usuario.ErrInvitacionExpirada) {
		t.Fatalf("esperaba expirada: %v", err)
	}
}

func TestVistaPreviaNoConsumeInvitacion(t *testing.T) {
	store := &equipoStore{preview: &usuario.Invitacion{Estado: "pendiente", ExpiresAt: time.Now().Add(time.Hour)}}
	svc := usuario.NuevoService(store, nil, nil)
	if _, err := svc.VistaPrevia(context.Background(), "token"); err != nil {
		t.Fatal(err)
	}
	if store.aceptada {
		t.Fatal("consultar el enlace no debe consumir la invitación")
	}
}

func TestVistaPreviaRechazaInvitacionesNoPendientes(t *testing.T) {
	for _, estado := range []string{"usada", "cancelada", "reemplazada"} {
		t.Run(estado, func(t *testing.T) {
			store := &equipoStore{preview: &usuario.Invitacion{Estado: estado, ExpiresAt: time.Now().Add(time.Hour)}}
			_, err := usuario.NuevoService(store, nil, nil).VistaPrevia(context.Background(), "token")
			if !errors.Is(err, usuario.ErrInvitacionNoValida) {
				t.Fatalf("estado %s debía rechazarse: %v", estado, err)
			}
		})
	}
}

func TestAceptarInvitacionConGoogleCreaAcceso(t *testing.T) {
	store := &equipoStore{preview: &usuario.Invitacion{Email: "persona@bar.com", Estado: "pendiente", ExpiresAt: time.Now().Add(time.Hour)}}
	authSvc := auth.NuevoServiceConGoogle(&authStore{}, nil, "client", &verifier{email: "persona@bar.com"})
	svc := usuario.NuevoService(store, authSvc, nil)

	if _, err := svc.Aceptar(context.Background(), "token", usuario.AceptarInvitacionInput{CredencialGoogle: "cred", Nombre: " Nombre Editado "}); err != nil {
		t.Fatal(err)
	}
	if !store.aceptada || store.nombreAceptado != "Nombre Editado" {
		t.Fatalf("la aceptación no llegó al alta atómica: aceptada=%v nombre=%q", store.aceptada, store.nombreAceptado)
	}
}
