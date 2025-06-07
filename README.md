# Worker-pool
## Что это
Это примитивный worker-pool с возможностью динамически добавлять и удалять воркеров. Может работать как на буфферизованных, так и на небуфферизованных каналах. 
Умеет в graceful shutdown.

## Examples
Примеры использования можно найти [здесь](./workerpool_test.go)

## Testing
```bash
git clone github.com/Kry0z1/workerpool
cd workerpool
go test -v -race .
```
