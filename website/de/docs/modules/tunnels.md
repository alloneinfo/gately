# Tailscale / frp / cloudflared

Diese drei Module binden die offiziellen Programme als verwaltete Prozesse ein: automatischer Neustart, in der Oberfläche gesammelte Logs und Konfiguration in der Oberfläche. Gately liefert diese Programme nicht mit, sodass Sie stets die neueste Upstream-Version nutzen können.

## Speicherort der Programme

Die ausführbaren Dateien werden in dieser Reihenfolge gesucht:

1. dem in den Moduleinstellungen angegebenen Pfad;
2. `PATH`;
3. dem Unterverzeichnis `bin/` des Konfigurationsverzeichnisses.

Legen Sie bei Docker-Bereitstellungen die Binärdatei in das eingebundene Konfigurationsverzeichnis, zum Beispiel `./gately-data/bin/frpc`. Tailscale benötigt sowohl `tailscale` als auch `tailscaled`.

Token und Schlüssel werden mit Rechten 0600 gespeichert. Das cloudflared-Token wird über eine Umgebungsvariable übergeben und erscheint daher nicht in der Prozessliste.

## Tailscale

- Verwaltet `tailscaled` (Userspace-Networking, containerfreundlich) oder bindet sich an einen vorhandenen System-Daemon an;
- Auth-Key oder Anmeldelink, Headscale wird unterstützt;
- Subnetz-Routen und Exit-Nodes;
- Knotenliste und SOCKS5-Proxy;
- Fester WireGuard-UDP-Port, der das Zulassen direkter Verbindungen durch eine Firewall erleichtert;
- Zusätzliche tailscaled-Argumente und Umgebungsvariablen. Nutzt iptables des Hosts das nf_tables-Backend, fügen Sie `TS_DEBUG_FIREWALL_MODE=nftables` zu den Umgebungsvariablen hinzu, sonst funktioniert die Subnetz-Weiterleitung nicht.

## frp

- **frpc-Client**: Proxys vom Typ tcp / udp / http / https / stcp / xtcp; Hinzufügen oder Entfernen wirkt sofort;
- **frps-Server**: Port-Allowlist, Subdomains, Dashboard;
- **frps-Lauschadressen**: „Lauschadresse“ legt fest, wo der frpc-Verbindungsport lauscht, „Proxy-Lauschadresse“, wo die HTTP- / HTTPS-Ports und die Remote-Ports der tcp- / udp-Proxys lauschen; leer bedeutet alle Adressen. Um frps hinter einen Reverse Proxy wie Gately zu stellen, die Lauschadresse auf `127.0.0.1` setzen, im Webdienst eine Site per Reverse Proxy auf `127.0.0.1:<Bind-Port>` leiten und als frpc-Transportprotokoll websocket oder wss zu dieser Site wählen; die Proxy-Lauschadresse leer lassen, damit die Remote-Ports der tcp-Proxys weiter von außen erreichbar sind. Beim Import einer frps-Konfiguration ohne `proxyBindAddr` wird wie bei frps `bindAddr` übernommen;
- **Bestehende Konfiguration importieren**: „Konfiguration importieren“ in der Seitenkopfzeile unterstützt ältere `frpc.ini` / `frps.ini` (einschließlich des Abschnitts `[common]`) und `toml` ab frp 0.52. Sie können eine Datei wählen oder den Inhalt einfügen. Sehen Sie sich zuerst das geparste Ergebnis und die ignorierten Felder in der Vorschau an und bestätigen Sie dann den Import. Beim Client können Sie wählen, ob angehängt (Proxys mit gleichem Namen bleiben unverändert) oder alle Proxys ersetzt werden. Der Import ändert den Aktivierungsstatus nicht und lässt sich auch über die API mit `POST /api/frp/import` aufrufen.

## Cloudflare Tunnel

- Token-Tunnel und temporäre trycloudflare-Tunnel;
- Mehrere Tunnel mit automatischem Neustart;
- Zeigt die öffentliche Adresse an.

## Die richtige Wahl

| Szenario | Empfehlung |
| --- | --- |
| Ein privates Netzwerk über mehrere Geräte aufbauen | Tailscale |
| Sie besitzen einen öffentlichen Server und wollen interne Dienste darüber nach außen abbilden | frp |
| Keine öffentliche IP und kein Server, aber ein Web-Dienst soll schnell erreichbar sein | Cloudflare Tunnel |
| Ihr Provider stellt ein NAT bereit, das Hole-Punching zulässt | [STUN-Traversal](/de/docs/modules/stun) |
