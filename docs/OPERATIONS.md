# Operação do Plate OCR 1.0

## Segurança

O plugin usa o token exclusivo criado pelo NVR em `/var/lib/nvr/plugin.token`. Ele não usa o token administrativo e não recebe as credenciais de câmera.

## Câmeras

Por padrão todas as câmeras habilitadas que possuem snapshot são elegíveis. `camera_ids` restringe a lista. Uma configuração por câmera pode definir ROI, faixa e direção.

## Taxa de análise

`poll_fps` controla a taxa-alvo. O snapshot HTTP é o transporte universal desta versão. Em câmeras com endpoint lento a taxa real será menor e aparecerá nas métricas.

## Spool

Quando o envio de evidência ou evento falha, o evento e sua evidência são persistidos em `spool_dir`. O runtime tenta reenviar periodicamente. O nome é derivado do `event_id`, mantendo idempotência.

## Métricas

- `plate_ocr_cameras`
- `plate_ocr_frames_fetched_total`
- `plate_ocr_frame_errors_total`
- `plate_ocr_events_published_total`
- `plate_ocr_events_spooled_total`
- `plate_ocr_spool_replayed_total`
- `plate_ocr_spool_pending`

## Ajuste inicial

1. Faça a placa ocupar pelo menos 70 px de largura.
2. Restrinja a ROI à pista útil.
3. Comece com `poll_fps=3`.
4. Mantenha `min_observations=2`.
5. Revise falsos positivos e falsos negativos de dia e à noite.
6. Só reduza thresholds depois de medir um dataset real.

## Evidência

O plugin envia o JPEG do melhor track. O NVR grava o arquivo com nome derivado do evento, calcula SHA-256 e devolve um `snapshot_ref`.

## Recuperação

Reiniciar o plugin não interrompe gravação nem live do NVR. Eventos consolidados e ainda não enviados permanecem no spool.
