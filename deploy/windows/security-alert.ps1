# Серверт халдлага, сэжигтэй үйлдэл илэрвэл имэйлээр анхааруулна (Task Scheduler-ээр 5 минут тутам, SYSTEM эрхээр).
#   🔴 Шууд: гаднаас амжилттай нэвтэрсэн (RDP, сүлжээгээр нууц үгтэй), шинэ хэрэглэгч / бүлэгт нэмсэн / нууц үг
#      сольсон, шинэ үйлчилгээ суусан (Google, Defender-ийн шинэчлэлээс бусад), лог устгасан, Defender вирус илрүүлсэн
#      эсвэл хамгаалалт унтарсан, surgalt үйлчилгээ зогссон.
#   🟠 Халдлагын огцом өсөлт (ангилал бүрт 6 цагт нэг удаа): 10 минутад Windows / SQL нууц үг таах оролдлого босгоос их.
#   📊 Өдөр бүр $DigestHour цагт тайлан: бүтэлгүй нэвтрэлт (IP-гаар), автоматаар блоклосон IP, surgalt.mn-ийн
#      нэвтрэх бүтэлгүй оролдлого, сканнер, 5xx.
# Имэйл: E:\new_www\surgalt\.env-ийн SMTP_* тохиргоогоор ALERT_EMAILS (байхгүй бол ADMIN_EMAILS) хаяг руу.
# Илгээж чадаагүй шууд анхааруулгыг дараагийн удаа дахин илгээнэ. Үйлдэл бүр alert.log-д.
#   Туршилт (илгээхгүй, төлөв хадгалахгүй): powershell -File security-alert.ps1 -DryRun [-SinceMinutes 60] [-Digest]
#   Туршилтын имэйл: powershell -File security-alert.ps1 -Test
param([switch]$DryRun, [switch]$Test, [switch]$Digest, [int]$SinceMinutes = 0)

$ErrorActionPreference = 'Stop'
$EnvFile = 'E:\new_www\surgalt\.env'
$Dir = 'C:\ProgramData\security-alert'
$StateFile = Join-Path $Dir 'state.json'
$LogFile = Join-Path $Dir 'alert.log'
$AutoBlockLog = 'C:\ProgramData\sql-autoblock\autoblock.log'
$IisLogDir = 'C:\inetpub\logs\LogFiles\W3SVC85' # surgalt.mn
$Service = 'surgalt'
$SelfIP = '103.48.116.81' # серверийн өөрийн IP
$DigestHour = 8
$WinSpike = 3000    # 10 минутад Windows-ын бүтэлгүй нэвтрэлт (одоогийн суурь түвшин ~1300)
$SqlSpike = 300     # 10 минутад SQL Server-ийн бүтэлгүй нэвтрэлт
$SpikeEveryH = 6
$SeenH = 6          # нэг хэрэглэгч, IP-ийн нэвтрэлтийг энэ хугацаанд дахин мэдэгдэхгүй
$BenignService = '\\Program Files( \(x86\))?\\Google\\(GoogleUpdater|Chrome Remote Desktop|Update|Drive File Stream)\\|\\Windows Defender\\|\\drivers\\wd\\|^"?C:\\Program Files\\Google\\Drive File Stream\\'

function Write-Log([string]$msg) {
    if (-not (Test-Path $Dir)) { New-Item -ItemType Directory -Force $Dir | Out-Null }
    Add-Content -Path $LogFile -Value ("{0}  {1}" -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $msg) -Encoding UTF8
}

function Test-PublicIP([string]$ip) {
    if (-not $ip -or $ip -eq '-' -or $ip -eq $SelfIP) { return $false }
    return $ip -notmatch '^(10\.|192\.168\.|172\.(1[6-9]|2[0-9]|3[01])\.|127\.|169\.254\.|0\.|fe80:|::1$|::$)'
}

function Utc([datetime]$t) { $t.ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ss.fffZ') }

function Get-Events([string]$log, [int[]]$ids, [datetime]$from, [datetime]$to, [string]$extra = '') {
    $idq = ($ids | ForEach-Object { "EventID=$_" }) -join ' or '
    $xp = "*[System[($idq) and TimeCreated[@SystemTime>='{0}' and @SystemTime<'{1}']]]{2}" -f (Utc $from), (Utc $to), $extra
    @(Get-WinEvent -LogName $log -FilterXPath $xp -ErrorAction SilentlyContinue)
}

function Get-Data($e) {
    $h = @{}
    foreach ($d in ([xml]$e.ToXml()).Event.EventData.Data) { $h[[string]$d.Name] = [string]$d.'#text' }
    $h
}

function Get-SidName([string]$sid) {
    try { (New-Object Security.Principal.SecurityIdentifier $sid).Translate([Security.Principal.NTAccount]).Value } catch { $sid }
}

function ConvertTo-Hash($o) {
    if ($null -eq $o) { return $null }
    if ($o -is [System.Management.Automation.PSCustomObject]) {
        $h = @{}
        foreach ($p in $o.PSObject.Properties) { $h[$p.Name] = ConvertTo-Hash $p.Value }
        return $h
    }
    if ($o -is [System.Collections.IEnumerable] -and $o -isnot [string]) { return , @($o | ForEach-Object { ConvertTo-Hash $_ }) }
    return $o
}

function New-Day([datetime]$t) { @{ since = $t.ToString('o'); win = @{}; sql = @{}; users = @{}; remote = @(); anon = 0 } }

function Read-State {
    $s = $null
    if (Test-Path $StateFile) { try { $s = ConvertTo-Hash (Get-Content $StateFile -Raw -Encoding UTF8 | ConvertFrom-Json) } catch { $s = $null } }
    if (-not $s) { $s = @{} }
    foreach ($k in 'seen', 'spike') { if (-not $s[$k]) { $s[$k] = @{} } }
    if (-not $s['pending']) { $s['pending'] = @() }
    if (-not $s['day']) { $s['day'] = New-Day (Get-Date) }
    foreach ($k in 'win', 'sql', 'users') { if (-not $s['day'][$k]) { $s['day'][$k] = @{} } }
    if (-not $s['day']['remote']) { $s['day']['remote'] = @() }
    $s
}

function Parse-Time([string]$s) { [datetime]::Parse($s, $null, [Globalization.DateTimeStyles]::RoundtripKind).ToLocalTime() }

function Add-Count($map, [string]$key, [int]$n = 1) {
    if (-not $key) { $key = '-' }
    if ($map.ContainsKey($key)) { $map[$key] = [int]$map[$key] + $n } else { $map[$key] = $n }
}

function Format-Top($map, [int]$n) {
    @($map.GetEnumerator() | Sort-Object { [int]$_.Value } -Descending | Select-Object -First $n | ForEach-Object { "  {0,9:N0}  {1}" -f [int]$_.Value, $_.Key })
}

function Get-Sum($map) { [int](($map.Values | ForEach-Object { [int]$_ } | Measure-Object -Sum).Sum) }

function Read-Env {
    $h = @{}
    foreach ($l in Get-Content $EnvFile -Encoding UTF8) {
        if ($l -match '^\s*([A-Za-z0-9_]+)\s*=\s*(.*)$') { $h[$Matches[1]] = $Matches[2].Trim().Trim('"').Trim("'") }
    }
    $h
}

function Send-Alert([string]$subject, [string]$body) {
    $subject = '[surgalt сервер] ' + $subject
    $body += "`r`n`r`n— $env:COMPUTERNAME · {0:yyyy-MM-dd HH:mm} (Улаанбаатар) · security-alert.ps1" -f (Get-Date)
    if ($DryRun) { Write-Host "===== $subject`r`n$body`r`n"; return }
    $e = Read-Env
    $user = $e['SMTP_USER']
    $pass = $e['SMTP_PASS'] -replace '\s', '' # Gmail App password зайтай байж болно
    $smtpHost = $e['SMTP_HOST']
    if (-not $smtpHost -and $user -match '@(gmail|googlemail)\.com$') { $smtpHost = 'smtp.gmail.com' }
    $port = 587 # 465 (шууд TLS)-ийг Send-MailMessage дэмждэггүй
    if ($e['SMTP_PORT'] -match '^\d+$' -and $e['SMTP_PORT'] -ne '465') { $port = [int]$e['SMTP_PORT'] }
    $from = $e['SMTP_FROM']
    if (-not $from) { $from = $user }
    $list = $e['ALERT_EMAILS']
    if (-not $list) { $list = $e['ADMIN_EMAILS'] }
    $to = @("$list" -split ',' | ForEach-Object { $_.Trim() } | Where-Object { $_ })
    if (-not $smtpHost -or -not $user -or -not $pass -or $to.Count -eq 0) { throw 'имэйлийн тохиргоо дутуу (.env: SMTP_USER, SMTP_PASS, ADMIN_EMAILS)' }
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    $cred = New-Object Management.Automation.PSCredential($user, (ConvertTo-SecureString $pass -AsPlainText -Force))
    Send-MailMessage -SmtpServer $smtpHost -Port $port -UseSsl -Credential $cred -From $from -To $to -Subject $subject -Body $body -Encoding UTF8
    Write-Log ("илгээв: " + $subject)
}

# ---- surgalt.mn-ийн IIS лог (сүүлийн 24 цаг): нэвтрэх бүтэлгүй, хэт олон оролдлого, сканнер, 5xx ----
function Get-WebStats([datetime]$from) {
    $r = @{ login401 = 0; login429 = 0; err5xx = 0; bad = @{}; scan = @{} }
    $cut = $from.ToUniversalTime().ToString('yyyy-MM-dd HH:mm:ss')
    foreach ($f in @(Get-ChildItem $IisLogDir -Filter 'u_ex*.log' -ErrorAction SilentlyContinue | Sort-Object Name | Select-Object -Last 2)) {
        $fs = New-Object IO.FileStream($f.FullName, [IO.FileMode]::Open, [IO.FileAccess]::Read, [IO.FileShare]::ReadWrite) # IIS бичиж байгаа
        $rd = New-Object IO.StreamReader($fs)
        while ($null -ne ($l = $rd.ReadLine())) {
            if ($l.StartsWith('#')) { continue }
            $p = $l.Split(' ')
            if ($p.Count -lt 12 -or ($p[0] + ' ' + $p[1]) -lt $cut) { continue }
            $uri, $ip, $code = $p[4], $p[8], $p[11]
            if ($uri -eq '/api/auth/login' -and $code -eq '401') { $r.login401++; Add-Count $r.bad $ip }
            elseif ($uri -eq '/api/auth/login' -and $code -eq '429') { $r.login429++; Add-Count $r.bad $ip }
            elseif ($code -eq '404' -and $uri -match '(?i)wp-|wordpress|\.php|\.env|\.git|phpmyadmin|/cgi-bin|\.asp|/admin\b|xmlrpc|\.sql|\.bak') { Add-Count $r.scan $ip }
            if ($code -match '^5\d\d$') { $r.err5xx++ }
        }
        $rd.Close()
    }
    $r
}

if ($Test) {
    Send-Alert '✅ Туршилтын анхааруулга' "Халдлагын анхааруулга ажиллаж байна. Гаднаас нэвтрэх, шинэ хэрэглэгч, шинэ үйлчилгээ, вирус, лог устгах, surgalt зогсох зэрэгт шууд, нууц үг таах халдлага огцом өсвөл 6 цагт нэг удаа, өдөр бүр $DigestHour`:00-д тайлан ирнэ."
    return
}

try {
    $now = Get-Date
    $st = Read-State
    $since = $now.AddMinutes(-10)
    if ($SinceMinutes -gt 0) { $since = $now.AddMinutes(-$SinceMinutes) } elseif ($st['last']) { $since = Parse-Time $st['last'] }
    if ($since -lt $now.AddHours(-24)) { $since = $now.AddHours(-24) }
    $crit = New-Object System.Collections.Generic.List[string]
    $day = $st['day']

    # Хуучирсан «мэдэгдсэн» тэмдэглэгээ
    foreach ($k in @($st['seen'].Keys)) { if ((Parse-Time $st['seen'][$k]) -lt $now.AddHours(-48)) { $st['seen'].Remove($k) } }

    # 1. Амжилттай нэвтрэлт: RDP (10) эсвэл сүлжээгээр (3) гаднын IP-аас.
    $xpLogon = " and *[EventData[Data[@Name='LogonType']='10' or Data[@Name='LogonType']='3']]"
    foreach ($e in (Get-Events 'Security' 4624 $since $now $xpLogon)) {
        $d = Get-Data $e
        $ip = $d['IpAddress'] -replace '^::ffff:', ''
        $rdp = $d['LogonType'] -eq '10'
        if (-not $rdp -and -not (Test-PublicIP $ip)) { continue }
        if ($d['TargetUserName'] -eq 'ANONYMOUS LOGON') { $day['anon'] = [int]$day['anon'] + 1; continue }
        $acct = $d['TargetDomainName'] + '\' + $d['TargetUserName']
        $kind = if ($rdp) { 'RDP' } else { 'сүлжээ (SMB/WinRM/SQL Windows auth), ' + $d['AuthenticationPackageName'] }
        $line = "{0:MM-dd HH:mm}  {1} ← {2}  [{3}]" -f $e.TimeCreated, $acct, $ip, $kind
        $key = "logon|$acct|$ip|$($d['LogonType'])"
        if ($st['seen'][$key] -and (Parse-Time $st['seen'][$key]) -gt $now.AddHours(-$SeenH)) { continue }
        $st['seen'][$key] = $now.ToString('o')
        $crit.Add("🔐 Гаднаас амжилттай нэвтэрлээ: $line")
        $day['remote'] = @($day['remote']) + $line
    }

    # 2. Хэрэглэгч, бүлгийн өөрчлөлт.
    foreach ($e in (Get-Events 'Security' 4720, 4722, 4724, 4728, 4732, 4756 $since $now)) {
        $d = Get-Data $e
        $by = $d['SubjectDomainName'] + '\' + $d['SubjectUserName']
        $t = "{0:MM-dd HH:mm}" -f $e.TimeCreated
        switch ($e.Id) {
            4720 { $crit.Add("👤 Шинэ хэрэглэгч үүсгэв: $($d['TargetUserName']) ($by, $t)") }
            4722 { $crit.Add("👤 Хэрэглэгчийг идэвхжүүлэв: $($d['TargetUserName']) ($by, $t)") }
            4724 { $crit.Add("🔑 Нууц үг сольсон (reset): $($d['TargetUserName']) ($by, $t)") }
            default { $crit.Add("👥 «$($d['TargetUserName'])» бүлэгт нэмэв: $(Get-SidName $d['MemberSid']) ($by, $t)") }
        }
    }

    # 3. Лог устгасан.
    foreach ($e in @(Get-Events 'Security' 1102 $since $now) + @(Get-Events 'System' 104 $since $now)) {
        $crit.Add(("🧹 Лог устгагдлаа ({0:MM-dd HH:mm}): {1}" -f $e.TimeCreated, (($e.Message -split "`n")[0]).Trim()))
    }

    # 3б. Chrome Remote Desktop-оор холбогдсон (1) эсвэл татгалзсан (3).
    foreach ($e in (Get-Events 'Application' 1, 3 $since $now " and *[System[Provider[@Name='chromoting']]]")) {
        $crit.Add(("🖥️ Chrome Remote Desktop: {0} ({1:MM-dd HH:mm})" -f ($e.Message -replace '\s+', ' ').Trim(), $e.TimeCreated))
    }

    # 4. Шинэ үйлчилгээ (Google, Defender-ийн шинэчлэлээс бусад).
    foreach ($e in (Get-Events 'System' 7045 $since $now)) {
        $d = Get-Data $e
        if ($d['ImagePath'] -match $BenignService) { continue }
        $crit.Add(("⚙️ Шинэ үйлчилгээ суулаа: {0} — {1} [{2}] ({3:MM-dd HH:mm})" -f $d['ServiceName'], $d['ImagePath'], $d['AccountName'], $e.TimeCreated))
    }

    # 5. Windows Defender: вирус илэрсэн, хамгаалалт унтарсан.
    foreach ($e in (Get-Events 'Microsoft-Windows-Windows Defender/Operational' 1006, 1015, 1116, 1117, 5001, 5010, 5012 $since $now)) {
        $d = Get-Data $e
        $what = switch ($e.Id) { 5001 { 'Бодит цагийн хамгаалалт УНТАРЛАА' } 5010 { 'Хортой программын хайлт унтарлаа' } 5012 { 'Вирусын хайлт унтарлаа' } default { 'Хортой программ илэрлээ' } }
        $crit.Add(("🦠 Defender: {0} {1} {2} ({3:MM-dd HH:mm})" -f $what, $d['Threat Name'], $d['Path'], $e.TimeCreated))
    }

    # 6. surgalt үйлчилгээ (дараалсан 2 шалгалтаар зогссон бол).
    $svc = Get-Service $Service -ErrorAction SilentlyContinue
    if ($svc -and $svc.Status -ne 'Running') {
        if ($st['svcDown'] -and -not $st['svcAlerted']) { $crit.Add("⛔ surgalt үйлчилгээ зогссон байна ($($svc.Status)) — surgalt.mn ажиллахгүй."); $st['svcAlerted'] = $true }
        $st['svcDown'] = $true
    } elseif ($svc) {
        if ($st['svcAlerted']) { $crit.Add('✅ surgalt үйлчилгээ дахин ажиллаж байна.') }
        $st['svcDown'] = $false; $st['svcAlerted'] = $false
    }

    # 7. Нууц үг таах оролдлого: өдрийн тоолуур + 10 минутын огцом өсөлт.
    $cntFrom = $since
    if ($cntFrom -lt $now.AddMinutes(-60)) { $cntFrom = $now.AddMinutes(-60) }
    $win10 = $now.AddMinutes(-10)
    $qFrom = if ($cntFrom -lt $win10) { $cntFrom } else { $win10 }
    $spikes = @()
    $w = @{}; $wn = 0
    foreach ($e in (Get-Events 'Security' 4625 $qFrom $now)) {
        $ip = [string]$e.Properties[19].Value -replace '^::ffff:', ''
        if ($e.TimeCreated -ge $cntFrom) { Add-Count $day['win'] $ip; Add-Count $day['users'] ([string]$e.Properties[5].Value) }
        if ($e.TimeCreated -ge $win10) { Add-Count $w $ip; $wn++ }
    }
    $s = @{}; $sn = 0
    foreach ($e in (Get-Events 'Application' 18456 $qFrom $now)) {
        $text = ($e.Properties | ForEach-Object { $_.Value }) -join ' '
        $ip = if ($text -match '\[CLIENT: ([0-9a-fA-F\.:]+)\]') { $Matches[1] } else { '-' }
        if ($e.TimeCreated -ge $cntFrom) { Add-Count $day['sql'] $ip }
        if ($e.TimeCreated -ge $win10) { Add-Count $s $ip; $sn++ }
    }
    foreach ($c in @(@{ k = 'win'; n = $wn; m = $w; lim = $WinSpike; name = 'Windows (RDP/SMB)' }, @{ k = 'sql'; n = $sn; m = $s; lim = $SqlSpike; name = 'SQL Server' })) {
        if ($c.n -lt $c.lim) { continue }
        $last = $st['spike'][$c.k]
        if ($last -and (Parse-Time $last) -gt $now.AddHours(-$SpikeEveryH)) { continue }
        $st['spike'][$c.k] = $now.ToString('o')
        $msg = @("🟠 $($c.name)-д нууц үг таах халдлага огцом өслөө: сүүлийн 10 минутад $('{0:N0}' -f $c.n) оролдлого (босго $('{0:N0}' -f $c.lim)). Их оролдсон IP:") + @(Format-Top $c.m 10)
        $spikes += , $msg
    }
    # Өдрийн тоолуур хэт томрохгүй (олон мянган IP).
    foreach ($k in 'win', 'sql', 'users') {
        if ($day[$k].Count -gt 6000) {
            $keep = @{}
            $day[$k].GetEnumerator() | Sort-Object { [int]$_.Value } -Descending | Select-Object -First 3000 | ForEach-Object { $keep[$_.Key] = $_.Value }
            $day[$k] = $keep
        }
    }

    # ---- Илгээх ----
    $items = @($st['pending']) + @($crit)
    if ($items.Count -gt 0) {
        $body = "Серверт анхаарах үйлдэл илэрлээ:`r`n`r`n" + ($items -join "`r`n") + "`r`n`r`nТа өөрөө хийгээгүй бол нэн даруй шалгана уу (нууц үг солих, IP блоклох)."
        $subj = '🔴 Анхааруулга: ' + ($items[0] -replace '^\S+\s', '' -replace ':.*$', '')
        if ($items.Count -gt 1) { $subj += " (+$($items.Count - 1))" }
        try { Send-Alert $subj $body; $st['pending'] = @() }
        catch { Write-Log ("илгээж чадсангүй: " + $_.Exception.Message); $st['pending'] = @($items | Select-Object -Last 200) }
    }
    foreach ($sp in $spikes) {
        try { Send-Alert '🟠 Нууц үг таах халдлага огцом өслөө' (($sp -join "`r`n") + "`r`n`r`nХамгаалалт: 445/139/1433/5985 портыг гаднаас хаах, RDP-г зөвхөн өөрийн IP-аас зөвшөөрөх.") }
        catch { Write-Log ("илгээж чадсангүй: " + $_.Exception.Message) }
    }

    # ---- Өдрийн тайлан ----
    $today = $now.ToString('yyyy-MM-dd')
    if (-not $st['last'] -and -not $st['digest']) { $st['digest'] = $today } # анх суулгахад хоосон тайлан илгээхгүй
    if ($Digest -or ($now.Hour -ge $DigestHour -and $st['digest'] -ne $today)) {
        $from = Parse-Time $day['since']
        $web = Get-WebStats ($now.AddHours(-24))
        $blocked = @()
        if (Test-Path $AutoBlockLog) {
            $cut = $from.ToString('yyyy-MM-dd HH:mm:ss')
            $blocked = @(Get-Content $AutoBlockLog -Encoding UTF8 | Where-Object { $_ -match '^\d{4}-\d\d-\d\d \d\d:\d\d:\d\d  блоклов' -and $_.Substring(0, 19) -ge $cut })
        }
        $fwIPs = 0
        foreach ($r in @(Get-NetFirewallRule -DisplayName 'SQL auto-block *' -ErrorAction SilentlyContinue) + @(Get-NetFirewallRule -DisplayName 'Block SQL brute-force IPs*' -ErrorAction SilentlyContinue)) {
            if ($r) { $fwIPs += @(($r | Get-NetFirewallAddressFilter).RemoteAddress).Count }
        }
        $b = New-Object System.Collections.Generic.List[string]
        $b.Add(("Хугацаа: {0:yyyy-MM-dd HH:mm} — {1:yyyy-MM-dd HH:mm}" -f $from, $now))
        $b.Add('')
        $b.Add(("🔐 Гаднаас амжилттай нэвтэрсэн: {0}" -f @($day['remote']).Count))
        foreach ($x in (@($day['remote']) | Select-Object -Last 20)) { $b.Add("  $x") }
        $b.Add('')
        $b.Add(("🪟 Windows (RDP/SMB) бүтэлгүй нэвтрэлт: {0:N0} ({1:N0} IP)" -f (Get-Sum $day['win']), $day['win'].Count))
        foreach ($x in (Format-Top $day['win'] 10)) { $b.Add($x) }
        $b.Add('  Оролдсон нэр:')
        foreach ($x in (Format-Top $day['users'] 5)) { $b.Add($x) }
        $b.Add(("  Нэргүй (null session) холболт: {0:N0}" -f [int]$day['anon']))
        $b.Add('')
        $b.Add(("🗄️ SQL Server бүтэлгүй нэвтрэлт: {0:N0} ({1:N0} IP)" -f (Get-Sum $day['sql']), $day['sql'].Count))
        foreach ($x in (Format-Top $day['sql'] 10)) { $b.Add($x) }
        $b.Add(("  Автоматаар блоклосон IP: {0} (нийт блоклосон: {1})" -f $blocked.Count, $fwIPs))
        foreach ($x in ($blocked | Select-Object -Last 15)) { $b.Add("  " + $x.Substring(21)) }
        $b.Add('')
        $b.Add('🌐 surgalt.mn (сүүлийн 24 цаг):')
        $b.Add(("  Нэвтрэх бүтэлгүй (буруу нууц үг): {0:N0}, хэт олон оролдлого (хаагдсан): {1:N0}, серверийн алдаа 5xx: {2:N0}" -f $web.login401, $web.login429, $web.err5xx))
        foreach ($x in (Format-Top $web.bad 5)) { $b.Add($x) }
        $b.Add(("  Сканнер (WordPress, .env, .php хайсан): {0:N0} хүсэлт, {1} IP — бүгд 404, аюулгүй" -f (Get-Sum $web.scan), $web.scan.Count))
        foreach ($x in (Format-Top $web.scan 5)) { $b.Add($x) }
        try {
            Send-Alert ("📊 Өдрийн тайлан: {0:N0} халдлагын оролдлого, {1} гаднаас нэвтэрсэн" -f ((Get-Sum $day['win']) + (Get-Sum $day['sql'])), @($day['remote']).Count) ($b -join "`r`n")
            $st['digest'] = $today
            $st['day'] = New-Day $now
        } catch { Write-Log ("тайлан илгээж чадсангүй: " + $_.Exception.Message) }
    }

    $st['last'] = $now.ToString('o')
    if (-not $DryRun) { $st | ConvertTo-Json -Depth 6 -Compress | Set-Content -Path $StateFile -Encoding UTF8 }
} catch {
    Write-Log ("алдаа: " + $_.Exception.Message)
    if ($DryRun) { throw }
}
