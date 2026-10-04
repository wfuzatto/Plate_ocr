# Plate OCR

Plugin oficial de OCR/LPR da plataforma NVR.

## Estado — 1.0.0

A versão 1.0 fecha o primeiro fluxo operacional completo e offline:

- recebe frames somente pelo NVR;
- usa token exclusivo de plugin, separado do token administrativo;
- provider clássico embutido, sem downloads e sem runtime externo;
- detector multi-escala por energia de bordas;
- OCR de 7 caracteres com templates 5x7;
- suporte aos formatos BR Mercosul e brasileiro anterior;
- leitura de placas em uma ou duas linhas;
- ROI, faixa e direção por câmera;
- normalização auditável de O/0, I/1, B/8, S/5, G/6 e Z/2;
- tracking e votação multi-frame;
- deduplicação temporal;
- evidência JPEG vinculada ao evento;
- spool em disco para indisponibilidade do NVR;
- healthcheck e métricas Prometheus;
- replay JSONL para regressão;
- binário Go sem dependências de rede em runtime.

O provider clássico é o baseline funcional. A interface Detector/Recognizer continua desacoplada para permitir um provider neural futuro sem alterar o protocolo, o evento ou o NVR.

## Fluxo

```text
camera
  -> NVR
  -> endpoint de frame do plugin
  -> plate_ocr
       -> ROI
       -> detector
       -> OCR
       -> normalização
       -> tracking multi-frame
       -> dedupe
       -> upload de evidência
       -> nvr.event.plate.detected.v1
  -> NVR event store / pesquisa
```

## Execução

```bash
./plate-ocr serve -config /etc/nvr/plugins/plate-ocr.json
```

Health: `http://127.0.0.1:8091/healthz`

Métricas: `http://127.0.0.1:8091/metrics`

## ROI por câmera

```json
{
  "cameras": {
    "camera-uuid": {
      "roi": {"x1": 0.10, "y1": 0.35, "x2": 0.95, "y2": 0.90},
      "lane": "faixa-1",
      "direction": "centro"
    }
  }
}
```

## Desenvolvimento completamente offline

```bash
make test
make build
make verify
```

O módulo Go não possui dependências externas. `GOPROXY=off` é usado em CI e na verificação local.

## Replay

```bash
./plate-ocr replay -config configs/default.json < observations.jsonl
```

## Limite do provider clássico

O OCR embutido é um baseline independente de Internet. Em alta velocidade, ângulo forte, motion blur, chuva, reflexo IR e placas muito pequenas, a precisão deve ser medida no dataset real. Um provider neural pode substituir detector e recognizer atrás das interfaces existentes sem alterar a integração.

Veja `docs/ARCHITECTURE.md`, `docs/EVENT_SCHEMA.md`, `docs/OPERATIONS.md` e `docs/VALIDATION.md`.
