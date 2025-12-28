# Как я написал Go-фреймворк с поддержкой REST, WebSocket и SSE в одном пакете

Привет, Хабр! В этой статье расскажу о своём опыте создания веб-фреймворка на Go, который объединяет HTTP, WebSocket и Server-Sent Events под одним API. Поделюсь техническими решениями, граблями, на которые наступил, и выводами, которые сделал.

<cut>

## Предыстория: почему не устроили существующие решения

Работая над проектом с real-time функциональностью, я столкнулся с типичной проблемой Go-разработчика: для полноценного бэкенда нужно собирать пазл из нескольких библиотек.

Типичный стек выглядит так:
- **Gin/Echo/Chi** — для REST API
- **Gorilla WebSocket** — для WebSocket
- **Самописное решение** — для SSE

Каждая библиотека имеет свой API, свои абстракции, свой подход к middleware. Когда нужно прокинуть контекст между HTTP-хендлером и WebSocket-соединением — начинаются костыли.

Я решил попробовать создать единый фреймворк, где все три протокола работают через общий Context. Назвал его Poltergeist (потому что призрак невидимо присутствует везде — как и мой фреймворк в каждом соединении).

## Архитектурные решения

### Единый Context

Первая задача — создать Context, который работает одинаково для HTTP, WebSocket и SSE.

В стандартном подходе для каждого типа соединения свой контекст:

```go
// Gin
func handler(c *gin.Context) {
    c.JSON(200, data)
}

// Gorilla WebSocket
func wsHandler(conn *websocket.Conn) {
    conn.WriteMessage(websocket.TextMessage, data)
}
```

Я хотел унифицировать это:

```go
type Context struct {
    Request  *http.Request
    Response http.ResponseWriter
    params   map[string]string
    store    map[string]interface{}
}
```

Ключевое решение — Context хранит только базовые HTTP-примитивы, а методы для работы с WebSocket/SSE добавляются через отдельные обёртки. Это позволяет:

1. Не раздувать основной Context
2. Переиспользовать middleware между протоколами
3. Иметь единый интерфейс для логирования и метрик

### Роутинг без рефлексии

Многие Go-фреймворки используют рефлексию для роутинга. Я пошёл другим путём — простое дерево с матчингом по сегментам пути:

```go
func (r *Router) matchPath(pattern, path string) (map[string]string, bool) {
    patternParts := splitPath(pattern)
    pathParts := splitPath(path)
    
    params := make(map[string]string)
    
    for i, part := range patternParts {
        if strings.HasPrefix(part, ":") {
            // Параметр пути
            params[part[1:]] = pathParts[i]
        } else if part != pathParts[i] {
            return nil, false
        }
    }
    
    return params, true
}
```

Это даёт предсказуемую производительность O(n), где n — количество сегментов пути. Для большинства API (5-10 сегментов) это быстрее, чем regexp-based решения.

### Управление WebSocket-соединениями: паттерн Hub

Одна из главных проблем WebSocket — управление множеством соединений. Gorilla WebSocket даёт низкоуровневый API, а дальше — сам разбирайся.

Я реализовал паттерн Hub, знакомый по Phoenix (Elixir) и Socket.IO:

```go
type Hub struct {
    connections map[string]Connection
    rooms       map[string]map[string]Connection
    mu          sync.RWMutex
}

func (h *Hub) BroadcastToRoom(room string, message []byte) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    for _, conn := range h.rooms[room] {
        conn.Send(message)
    }
}
```

Ключевые решения:
- **RWMutex вместо Mutex** — чтение (broadcast) происходит чаще записи (register/unregister)
- **Буферизованные каналы для отправки** — чтобы медленный клиент не блокировал broadcast
- **Graceful disconnect** — при отключении клиента не паникуем, а корректно удаляем из всех комнат

### SSE: проще, чем кажется

Server-Sent Events часто недооценивают. Для односторонней передачи данных (уведомления, live-ленты) SSE проще WebSocket:

- Работает через обычный HTTP
- Автоматический реконнект в браузере
- Не нужен отдельный протокол

Реализация оказалась тривиальной:

```go
type SSEWriter struct {
    w       http.ResponseWriter
    flusher http.Flusher
}

func (s *SSEWriter) Send(event SSEEvent) error {
    fmt.Fprintf(s.w, "event: %s\n", event.Event)
    fmt.Fprintf(s.w, "data: %s\n\n", event.Data)
    s.flusher.Flush()
    return nil
}
```

Важный нюанс: нужно проверять, поддерживает ли `ResponseWriter` интерфейс `Flusher`. Не все реализации это делают (например, некоторые прокси).

## Автогенерация Swagger: без аннотаций

Одна из болей Go-разработки — Swagger-документация. Стандартный подход (swaggo) требует писать комментарии:

```go
// @Summary Create user
// @Description Create a new user
// @Tags users
// @Accept json
// @Produce json
// @Param user body User true "User object"
// @Success 201 {object} User
// @Router /users [post]
func CreateUser(c *gin.Context) { ... }
```

Я решил генерировать OpenAPI из метаданных роута:

```go
app.Route(RouteConfig{
    Method:      "POST",
    Path:        "/users",
    Handler:     createUser,
    Description: "Создаёт нового пользователя",
    RequestBody: User{},
    Responses: map[int]interface{}{
        201: User{},
        400: ErrorResponse{},
    },
})
```

Под капотом — рефлексия для анализа структур и генерации JSON Schema:

```go
func generateSchema(t reflect.Type) Schema {
    schema := Schema{Type: "object", Properties: map[string]Schema{}}
    
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        jsonTag := field.Tag.Get("json")
        schema.Properties[jsonTag] = typeToSchema(field.Type)
    }
    
    return schema
}
```

Это работает для простых случаев. Для сложных (вложенные структуры, generics) пришлось добавить обработку рекурсии и кеширование схем.

## Бенчмарки: что получилось

После оптимизаций провёл бенчмарки:

| Операция | ns/op | B/op | allocs/op |
|----------|-------|------|-----------|
| JSON Response | 197 | 48 | 1 |
| Path Param | 14 | 0 | 0 |
| Static Route | 168 | 360 | 2 |
| Dynamic Route | 238 | 368 | 3 |

Для сравнения с Gin запустил нагрузочное тестирование:

| Фреймворк | req/sec | p99 latency |
|-----------|---------|-------------|
| Poltergeist | 127,000 | 0.92ms |
| Gin | 118,000 | 1.12ms |
| Fiber | 142,000 | 0.78ms |

Fiber быстрее за счёт fasthttp, но у него свои ограничения (не совместим со стандартной библиотекой). Для моих задач производительность Poltergeist достаточная.

## На какие грабли наступил

### 1. Гонки при broadcast

Первая версия Hub падала под нагрузкой. Причина — одновременная запись в map при регистрации и чтение при broadcast.

Решение: RWMutex + копирование слайса соединений перед отправкой.

### 2. Утечка горутин в SSE

Если клиент отключался, горутина отправки зависала на записи в закрытое соединение.

Решение: select с context.Done():

```go
select {
case <-ctx.Done():
    return
case msg := <-messages:
    writer.Send(msg)
}
```

### 3. Middleware и порядок вызова

Изначально middleware вызывались в порядке добавления. Но для Recovery middleware это неправильно — он должен быть первым, чтобы поймать панику из любого места.

Решение: добавил приоритеты middleware и сортировку.

## Выводы

1. **Единый Context для разных протоколов** — рабочая идея, упрощает код
2. **Hub-паттерн** — must-have для WebSocket-приложений
3. **SSE недооценён** — для многих задач проще WebSocket
4. **Автогенерация документации** — экономит время, но рефлексия имеет ограничения

Фреймворк получился нишевым: он не заменит Gin для чистого REST API, но удобен для проектов с real-time функциональностью.

Исходный код открыт: [github.com/gofuckbiz/poltergeist](https://github.com/gofuckbiz/poltergeist)

Буду рад обратной связи и pull-реквестам. Если есть вопросы по реализации — спрашивайте в комментариях.

---

**Теги:** Go, Golang, WebSocket, SSE, веб-разработка

**Хабы:** Go, Разработка веб-сайтов, Open source
