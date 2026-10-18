# WAF-Firewall

Eine Web Application Firewall auf Basis von [Coraza](https://coraza.io/) und OWASP CRS v4, die sich Sites in den [Web-Diensten](/de/docs/modules/web) zuordnen lässt.

## Funktionen

- Integriertes OWASP-CRS-v4-Regelwerk, dessen Regeldateien einzeln aktiviert werden können;
- Eigene SecLang-Regeln;
- Blockiermodus und reiner Erkennungsmodus;
- Mehrere Instanzen, auf die Sites bei Bedarf verweisen.

## Empfehlungen

1. Starten Sie bei einer neuen Site mit „Nur erkennen“ und beobachten Sie die Logs eine Weile, um Fehlalarme auszuschließen;
2. Passen Sie bei Fehlalarmen die Regeldateien an oder fügen Sie eigene Ausnahmen hinzu;
3. Wechseln Sie danach auf „Blockieren“.

## Verifizierung

Die WAF wird mit echten Angriffs-Payloads getestet und deckt die Blockierung von Angriffen wie Path Traversal ab.
