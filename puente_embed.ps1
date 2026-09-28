# Trollhunters command bridge (embedded)
# 1. Abre PowerShell y ejecuta este archivo:
#      powershell -ExecutionPolicy Bypass -File "$env:USERPROFILE\Desktop\puente.ps1"
#    (o copia y pega todo el contenido en una ventana de PowerShell)
# 2. Deja esa ventana ABIERTA.
# 3. En OTRA ventana de PowerShell, abre el tunel HTTP:
#      cd ([Environment]::GetFolderPath("Desktop"))
#      .\cloudflared-windows-amd64.exe tunnel --url http://localhost:8080
# 4. Pega en el chat la direccion https://....trycloudflare.com que aparezca.
# 5. Deja AMBAS ventanas abiertas.

$token = "CHANGE-ME-USE-A-LONG-RANDOM-TOKEN"
$port = 8080

$listener = New-Object System.Net.HttpListener
$listener.Prefixes.Add("http://localhost:$port/")
try {
    $listener.Start()
} catch {
    Write-Host "ERROR al iniciar el puente: $($_.Exception.Message)"
    Write-Host "Presiona Enter para salir..."
    Read-Host
    exit 1
}

Write-Host ""
Write-Host "Puente activo en http://localhost:$port/"
Write-Host "NO cierres esta ventana."
Write-Host ""

while ($listener.IsListening) {
    $ctx = $listener.GetContext()
    $req = $ctx.Request
    $res = $ctx.Response
    $res.ContentType = "text/plain; charset=utf-8"
    try {
        if ($req.Headers["X-Token"] -ne $token) {
            $res.StatusCode = 403
            $msg = "token invalido"
        } else {
            $reader = New-Object System.IO.StreamReader($req.InputStream, [Text.Encoding]::UTF8)
            $cmd = $reader.ReadToEnd()
            $reader.Close()
            if ([string]::IsNullOrWhiteSpace($cmd)) {
                $msg = "(comando vacio)"
            } else {
                Write-Host ">> $cmd"
                $msg = Invoke-Expression $cmd 2>&1 | Out-String
                if ([string]::IsNullOrEmpty($msg)) { $msg = "(sin salida)" }
            }
        }
    } catch {
        $msg = "ERROR: $($_.Exception.Message)"
    }
    $bytes = [Text.Encoding]::UTF8.GetBytes($msg)
    $res.ContentLength64 = $bytes.Length
    $res.OutputStream.Write($bytes, 0, $bytes.Length)
    $res.Close()
}
