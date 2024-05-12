# API-сервер для удаленного запуска bash скриптов

## Начало работы

Чтобы настроить API-сервер и начать использование, выполните следующие шаги:

1. Запуск:
   ```bash
   make compose-up
   ```
1. Миграции:
   ```bash
   make test-migration-up
   ```
2. Соберите и запустите API-сервер:
   ```bash
   make
   ./app
   ```

API-сервер начнет работу на указанном порту (по умолчанию: 8080).

## API-точки

### Создание новой команды.

Создает новый сегмент с указанным идентификатором.

**Точка входа:** `POST /commands`

#### Пример запроса

```bash
curl --location 'localhost:8080/commands' \
--header 'Content-Type: application/json' \
--data '{
    "script": "echo Hello World"
}'
```

#### Пример ответа

```json
{"id":149,"script":"echo Hello World","status":"completed","output":"Hello World\n","created_at":"2024-05-12T11:37:27.007995Z"}
```
#### Так же можно передавать несколько команд через ;
#### Пример запроса

```bash
curl --location 'localhost:8080/commands' \
--header 'Content-Type: application/json' \
--data '{
    "script": "echo Hello World; sleep 5; echo Bye"
}'
```

#### Тогда в ответ придёт информация о том, что запрос запущен, а информацию о выполнении можно увидеть по id
А в базу данных постепенно будет добавляться информация о выводе

```json
{"id":150,"script":"echo Hello World; sleep 5; echo Bye","status":"in process","output":"","created_at":"2024-05-12T11:39:12.80096Z"}
```

Если одиночная команда выполняется дольше 3 секунд, то сервер возвращает ответ только о начале процесса, а если меньше то сразу и вывод консоли


### Получение списка команд

Получение всех ранее выполненных команд, хранящихся в базе данных

**Точка входа:** `GET /commands`

#### Пример запроса

```bash
curl --location 'localhost:8080/commands' \
--data ''
```

#### Пример ответа

```json
{"id":153,"script":"echo Hello World; sleep 5; echo Bye","status":"completed","output":"Hello World\nBye\n","created_at":"2024-05-12T11:45:27.710209Z"},
{"id":154,"script":"echo Hello World","status":"completed","output":"Hello World\n","created_at":"2024-05-12T11:45:33.417569Z"}
```

### Получение одной команды

Получение ранее выполненной команды по id

**Точка входа:** `GET /commands/{id}`

#### Пример запроса

```bash
curl --location 'localhost:8080/commands/154' \
--data ''
```

#### Пример ответа

```json
{"id":154,"script":"echo Hello World","status":"completed","output":"Hello World\n","created_at":"2024-05-12T11:45:33.417569Z"}
```




### Остановка команды
Запрос останавливает выполнение ещё не завершенной команды по id
Если была послана одна команда, то запрос останавливает эту команду. Если команд было несколько, то запрос останавливает текущую и все последующие

**Точка входа:** `DELETE /commands/{id}`

#### Пример запроса

```bash
curl --location --request DELETE 'localhost:8080/commands/162'
```

#### Пример ответа

```json
"Commands stopped"
```

