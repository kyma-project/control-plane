package templates

const (
	FluentBitExtraConf = `
[INPUT]
    Name              tail
    Tag               dex.*
    Path              /var/log/containers/*_dex-*.log
    DB                /var/log/flb_kube_dex.db
    parser            docker
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On
    Refresh_Interval  10
[FILTER]
    Name    lua
    Match   dex.*
    script  script.lua
    call    reformat
[FILTER]
    Name    grep
    Match   dex.*
    Regex   time .*
[FILTER]
    Name    grep
    Match   dex.*
    Regex   data .*\"xsuaa
[OUTPUT]
    Name             {{.HttpPlugin}}
    Match            dex.*
    Retry_Limit      False
    Host             {{.Host}}
    Port             {{.Port}}
    URI              {{.Path}}security-events
    Header           Content-Type application/json
    HTTP_User        {{.Config.User}}
    HTTP_Passwd      {{.Config.Password}}
    Format           json_stream
    tls              on`
)
