// Conector Trollhunters v2: con doble clic levanta el puente de comandos y el
// tunel de Cloudflare, y muestra la URL lista para pegar en el chat.
// Novedad v2: si el túnel se cae, se reconecta SOLO y copia la nueva URL
// al portapapeles (ya no hay que abrir el programa de nuevo).
//
// El puente PowerShell va embebido en el .exe (puente_embed.ps1). El token de
// seguridad del puente debe coincidir con el de pc-cmd.sh en la maquina del
// asistente; si algun dia cambia, hay que recompilar este programa.
package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

//go:embed puente_embed.ps1
var puentePS1 string

const bridgePort = 8080

// URL oficial de descarga de cloudflared para Windows (armada por partes).
var cloudflaredURL = "https://github.com/cloudflare/cloudflared/releases/" +
	"latest/download/" + "cloudflared-windows-amd64.exe"

var tunnelRe = regexp.MustCompile("https://[a-z0-9-]+\\.trycloudflare\\.com")

func workDir() string {
	d := filepath.Join(os.Getenv("LOCALAPPDATA"), "Trollhunters")
	_ = os.MkdirAll(d, 0755)
	return d
}

func portOpen() bool {
	c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", bridgePort), 2*time.Second)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

// startBridge abre el puente en su propia ventana visible de PowerShell.
func startBridge(ps1Path string) {
	cmd := exec.Command("cmd", "/c", "start", "Puente Trollhunters - NO cerrar",
		"powershell", "-NoExit", "-ExecutionPolicy", "Bypass", "-File", ps1Path)
	_ = cmd.Run()
}

// killStaleCloudflared elimina tuneles huerfanos de ejecuciones anteriores.
func killStaleCloudflared() {
	cmd := exec.Command("taskkill", "/F", "/IM", "cloudflared.exe")
	_ = cmd.Run()
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// ensureCloudflared busca cloudflared.exe junto al .exe o en la carpeta de
// trabajo; si no existe lo descarga una sola vez desde la pagina oficial.
func ensureCloudflared(dir string) (string, error) {
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "cloudflared.exe")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	p := filepath.Join(dir, "cloudflared.exe")
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	fmt.Println("Descargando cloudflared oficial (solo la primera vez)...")
	if err := downloadFile(cloudflaredURL, p); err != nil {
		return "", err
	}
	fmt.Println("Descarga lista.")
	return p, nil
}

// extractTunnelURL lee la salida de cloudflared hasta encontrar la URL del
// tunel o hasta agotar el tiempo limite.
func extractTunnelURL(r io.Reader, timeout time.Duration) (string, error) {
	found := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			if m := tunnelRe.FindString(scanner.Text()); m != "" {
				found <- m
				return
			}
		}
	}()
	select {
	case u := <-found:
		return u, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("no se encontró la URL del túnel")
	}
}

// runTunnelLoop mantiene el túnel vivo: si cloudflared se cae, espera 5
// segundos y lo relanza solo, copiando la nueva URL al portapapeles.
// También revive el puente si dejó de responder.
func runTunnelLoop(cf, local, ps1Path string) {
	hostHeader := fmt.Sprintf("localhost:%d", bridgePort)
	reconnects := 0
	for {
		if !portOpen() {
			fmt.Println("[AVISO] El puente no responde. Reiniciándolo...")
			startBridge(ps1Path)
			for i := 0; i < 30; i++ {
				time.Sleep(time.Second)
				if portOpen() {
					break
				}
			}
		}
		killStaleCloudflared()
		if reconnects > 0 {
			fmt.Println("")
			fmt.Println("Reabriendo el túnel (intento", reconnects+1, ")...")
		} else {
			fmt.Println("Abriendo el túnel (toma unos segundos)...")
		}
		cmd := exec.Command(cf, "tunnel", "--url", local,
			"--http-host-header", hostHeader)
		out, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Println("[ERROR]", err)
			time.Sleep(5 * time.Second)
			reconnects++
			continue
		}
		cmd.Stderr = cmd.Stdout // unir salidas: la URL puede salir por cualquiera
		if err := cmd.Start(); err != nil {
			fmt.Println("[ERROR]", err)
			time.Sleep(5 * time.Second)
			reconnects++
			continue
		}
		url, err := extractTunnelURL(out, 90*time.Second)
		if err != nil {
			fmt.Println("[ERROR] No se encontró la URL del túnel. Reintentando...")
			_ = cmd.Process.Kill()
			time.Sleep(5 * time.Second)
			reconnects++
			continue
		}
		copyToClipboard(url)
		fmt.Println("")
		if reconnects > 0 {
			fmt.Println("  NUEVA URL (ya copiada al portapapeles):")
		} else {
			fmt.Println("  TU URL ES (ya copiada al portapapeles):")
		}
		fmt.Println("")
		fmt.Println("   ", url)
		fmt.Println("")
		if reconnects == 0 {
			fmt.Println("  Pégala en el chat de Trollhunters.")
			fmt.Println("")
			fmt.Println("  Deja ESTA ventana y la del puente ABIERTAS")
			fmt.Println("  mientras trabajamos. Para desconectar, ciérralas.")
			fmt.Println("  Si el túnel se cae, se reconecta SOLO.")
		} else {
			fmt.Println("  Si la URL cambió, pégala de nuevo en el chat.")
		}
		_ = cmd.Wait() // bloquea hasta que el túnel muera
		fmt.Println("")
		fmt.Println("[AVISO] El túnel se cayó. Reconectando solo en 5 segundos...")
		reconnects++
		time.Sleep(5 * time.Second)
	}
}

func copyToClipboard(s string) {
	cmd := exec.Command("cmd", "/c", "clip")
	cmd.Stdin = strings.NewReader(s)
	_ = cmd.Run()
}

func pause() {
	fmt.Println("\nPresiona Enter para salir...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

func main() {
	fmt.Println("==============================================")
	fmt.Println("  CONECTOR TROLLHUNTERS")
	fmt.Println("==============================================")
	fmt.Println("")

	dir := workDir()
	ps1Path := filepath.Join(dir, "puente.ps1")
	if err := os.WriteFile(ps1Path, []byte(puentePS1), 0644); err != nil {
		fmt.Println("[ERROR] No pude preparar el puente:", err)
		pause()
		return
	}

	killStaleCloudflared()

	if portOpen() {
		fmt.Println("[OK] El puente ya estaba activo.")
	} else {
		fmt.Println("Iniciando el puente (se abrirá una ventana)...")
		startBridge(ps1Path)
		ok := false
		for i := 0; i < 30; i++ {
			time.Sleep(time.Second)
			if portOpen() {
				ok = true
				break
			}
		}
		if !ok {
			fmt.Println("[ERROR] El puente no respondió. Revisa la ventana del puente.")
			pause()
			return
		}
		fmt.Println("[OK] Puente activo.")
	}

	cf, err := ensureCloudflared(dir)
	if err != nil {
		fmt.Println("[ERROR] No se pudo obtener cloudflared:", err)
		pause()
		return
	}

	local := fmt.Sprintf("http://localhost:%d", bridgePort)
	runTunnelLoop(cf, local, ps1Path)
}
