$FLUENTD_CONF = if ($env:FLUENTD_CONF) { $env:FLUENTD_CONF } else { "fluent.conf" };
$DEFAULT_FLUENT_CONF = "/etc/fluent-conf/${FLUENTD_CONF}";
$FLUENTD_OPT = $env:FLUENTD_OPT

echo "Using configuration: ${DEFAULT_FLUENT_CONF}"
echo "With FLUENTD_OPT: ${FLUENTD_OPT}"

$retries = if ($env:RETRIES) { $env:RETRIES } else { 60 };
$ready = $false

foreach ($attempt in 1..$retries) {
    if (Test-Path -Path $DEFAULT_FLUENT_CONF -PathType Leaf) {
        $ready = $true
        break
    }
    echo "Waiting for config file to become available: $attempt of $retries"
    Start-Sleep -Seconds 10
}
if ($ready -ne $true ) {return 1}
echo "Found configuration file: ${DEFAULT_FLUENT_CONF}"
$cmdline_args = "& fluentd", "-c", ${DEFAULT_FLUENT_CONF}, "-p", "/etc/fluent-plugin", $FLUENTD_OPT
$cmdline = $cmdline_args -join ' '
echo "cmdline $cmdline"
Invoke-Expression $cmdline
#& \fluentd.cmd -c ${DEFAULT_FLUENT_CONF} -p /etc/fluent/plugins $FLUENTD_OPT
