# SQL Server-т нууц үг таах халдлагаас автомат хамгаалалт (Task Scheduler-ээр минут тутам, SYSTEM эрхээр).
# Windows Application log-ийн 18456 (Login failed … [CLIENT: ip]) үйл явдлаас сүүлийн $WindowMin минутад $Threshold
# буюу түүнээс олон удаа бүтэлгүйтсэн IP-г Windows Firewall-ийн «SQL auto-block N» дүрэмд нэмж бүх портоор блоклоно.
# Өөрийн сервер, localhost, дотоод сүлжээний хаягийг хэзээ ч блоклохгүй. Үйлдэл бүрийг autoblock.log-д бичнэ.
#   Туршилт: powershell -File sql-autoblock.ps1 -DryRun
#   Блокоос гаргах: Set-NetFirewallRule -DisplayName "SQL auto-block 1" -RemoteAddress (…хаягийг хассан жагсаалт…)
param([switch]$DryRun)

$ErrorActionPreference = 'Stop'
$Dir = 'C:\ProgramData\sql-autoblock'
$LogFile = Join-Path $Dir 'autoblock.log'
$Threshold = 5
$WindowMin = 10
$RulePrefix = 'SQL auto-block'
$PerRule = 400 # нэг дүрэмд хамгийн ихдээ
$Allow = @('127.0.0.1', '::1', '103.48.116.81')

function Write-Log([string]$msg) {
    if (-not (Test-Path $Dir)) { New-Item -ItemType Directory -Force $Dir | Out-Null }
    Add-Content -Path $LogFile -Value ("{0}  {1}" -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $msg) -Encoding UTF8
}

function Test-PrivateIP([string]$ip) {
    return $ip -match '^(10\.|192\.168\.|172\.(1[6-9]|2[0-9]|3[01])\.|127\.|169\.254\.|192\.0\.2\.|198\.51\.100\.|203\.0\.113\.|fe80:|::1$)' # дотоод, TEST-NET
}

try {
    $events = Get-WinEvent -FilterHashtable @{ LogName = 'Application'; Id = 18456; StartTime = (Get-Date).AddMinutes(-$WindowMin) } -ErrorAction SilentlyContinue
    $counts = @{}
    foreach ($e in $events) {
        $text = $e.Message
        if (-not $text) { $text = ($e.Properties | ForEach-Object { $_.Value }) -join ' ' } # мессеж файлгүй үед
        if ($text -match '\[CLIENT: ([0-9a-fA-F\.:]+)\]') { $counts[$Matches[1]] = 1 + [int]$counts[$Matches[1]] }
    }
    $bad = @($counts.GetEnumerator() | Where-Object { $_.Value -ge $Threshold -and $Allow -notcontains $_.Key -and -not (Test-PrivateIP $_.Key) })
    if ($bad.Count -eq 0) { return }

    # Аль хэдийн блоклогдсон хаягууд (автомат болон гараар үүсгэсэн дүрмүүд).
    $blocked = @{}
    $rules = @(Get-NetFirewallRule -DisplayName "$RulePrefix *" -ErrorAction SilentlyContinue)
    foreach ($r in @($rules) + @(Get-NetFirewallRule -DisplayName 'Block SQL brute-force IPs*' -ErrorAction SilentlyContinue)) {
        if ($r) { foreach ($a in @(($r | Get-NetFirewallAddressFilter).RemoteAddress)) { $blocked[$a] = $true } }
    }
    $new = @($bad | Where-Object { -not $blocked[$_.Key] })
    if ($new.Count -eq 0) { return }

    foreach ($n in $new) {
        $ip = $n.Key
        if ($DryRun) { Write-Output ("[туршилт] блоклох: {0} ({1} оролдлого / {2} мин)" -f $ip, $n.Value, $WindowMin); continue }
        # Сул зайтай дүрэм эсвэл шинэ дүрэм.
        $target = $null
        foreach ($r in $rules) {
            $addr = @(($r | Get-NetFirewallAddressFilter).RemoteAddress)
            if ($addr.Count -lt $PerRule) { $target = $r; break }
        }
        if ($target) {
            $addr = @(($target | Get-NetFirewallAddressFilter).RemoteAddress) + $ip
            Set-NetFirewallRule -Name $target.Name -RemoteAddress $addr
        } else {
            $name = "$RulePrefix " + ($rules.Count + 1)
            $target = New-NetFirewallRule -DisplayName $name -Description 'SQL Server нууц үг таах халдлага (sql-autoblock.ps1)' -Direction Inbound -Action Block -Profile Any -RemoteAddress $ip
            $rules = @($rules) + $target
        }
        Write-Log ("блоклов {0} — {1} бүтэлгүй нэвтрэлт / {2} мин ({3})" -f $ip, $n.Value, $WindowMin, $target.DisplayName)
    }
} catch {
    Write-Log ("алдаа: " + $_.Exception.Message)
}
