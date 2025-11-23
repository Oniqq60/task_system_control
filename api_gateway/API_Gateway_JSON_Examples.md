# API Gateway - Примеры JSON запросов

## 🔐 Auth Endpoints

### 1. Register - Admin
**POST** `{{base_url}}/auth/register`

```json
{
    "name": "Admin User",
    "email": "admin@example.com",
    "password": "SecurePassword123!",
    "role": "admin"
}
```

**Response (201):**
```json
{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "admin@example.com",
    "name": "Admin User",
    "role": "admin"
}
```

---

### 2. Register - Employee
**POST** `{{base_url}}/auth/register`

```json
{
    "name": "Employee User",
    "email": "employee@example.com",
    "password": "SecurePassword123!",
    "role": "employee",
    "manager_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response (201):**
```json
{
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "email": "employee@example.com",
    "name": "Employee User",
    "role": "employee"
}
```

---

### 3. Login
**POST** `{{base_url}}/auth/login`

```json
{
    "email": "admin@example.com",
    "password": "SecurePassword123!"
}
```

**Response (200):**
```json
{
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "role": "admin"
}
```

---

### 4. Validate Token
**POST** `{{base_url}}/auth/validate`

**Вариант 1: Токен в body**
```json
{
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Вариант 2: Токен в Authorization header**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response (200):**
```json
{
    "valid": true,
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "role": "admin"
}
```

---

## 📋 Task Endpoints

### 1. Create Task
**POST** `{{base_url}}/task`  
**Headers:** `Authorization: Bearer {{admin_token}}`

```json
{
    "message": "Выполнить анализ данных за Q1 2025",
    "worker_id": "660e8400-e29b-41d4-a716-446655440001"
}
```

**Response (201):**
```json
{
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "message": "Выполнить анализ данных за Q1 2025",
    "status": "IN_PROGRESS",
    "worker_id": "660e8400-e29b-41d4-a716-446655440001",
    "created_by": "550e8400-e29b-41d4-a716-446655440000"
}
```

---

### 2. Update Task - Change Status to IN_PROGRESS
**PATCH** `{{base_url}}/task/{{task_id}}`  
**Headers:** `Authorization: Bearer {{employee_token}}`

```json
{
    "status": "IN_PROGRESS"
}
```

**Response (200):**
```json
{
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "message": "Выполнить анализ данных за Q1 2025",
    "status": "IN_PROGRESS",
    "worker_id": "660e8400-e29b-41d4-a716-446655440001"
}
```

---

### 3. Update Task - Change Status to NEEDS_HELP
**PATCH** `{{base_url}}/task/{{task_id}}`  
**Headers:** `Authorization: Bearer {{employee_token}}`

```json
{
    "status": "NEEDS_HELP",
    "reason": "Не хватает данных для анализа. Нужен доступ к базе данных за март 2025."
}
```

**Response (200):**
```json
{
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "message": "Выполнить анализ данных за Q1 2025",
    "status": "NEEDS_HELP",
    "worker_id": "660e8400-e29b-41d4-a716-446655440001",
    "reason": "Не хватает данных для анализа. Нужен доступ к базе данных за март 2025."
}
```

---

### 4. Update Task - Change Status to COMPLETED
**PATCH** `{{base_url}}/task/{{task_id}}`  
**Headers:** `Authorization: Bearer {{employee_token}}`

```json
{
    "status": "COMPLETED"
}
```

**Response (200):**
```json
{
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "message": "Выполнить анализ данных за Q1 2025",
    "status": "COMPLETED",
    "worker_id": "660e8400-e29b-41d4-a716-446655440001"
}
```

---

### 5. Update Task - Change Message
**PATCH** `{{base_url}}/task/{{task_id}}`  
**Headers:** `Authorization: Bearer {{admin_token}}`

```json
{
    "message": "Выполнить расширенный анализ данных за Q1 2025 с учетом новых требований"
}
```

**Response (200):**
```json
{
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "message": "Выполнить расширенный анализ данных за Q1 2025 с учетом новых требований",
    "status": "IN_PROGRESS",
    "worker_id": "660e8400-e29b-41d4-a716-446655440001"
}
```

---

### 6. Get Task List
**GET** `{{base_url}}/task`  
**Headers:** `Authorization: Bearer {{admin_token}}`

**Query Parameters:**
- `worker_id` (optional) - UUID сотрудника
- `created_by` (optional) - UUID создателя
- `status` (optional) - `IN_PROGRESS`, `COMPLETED`, `NEEDS_HELP`

**Примеры запросов:**

Все задачи:
```
GET {{base_url}}/task
```

Задачи конкретного сотрудника:
```
GET {{base_url}}/task?worker_id=660e8400-e29b-41d4-a716-446655440001
```

Задачи по статусу:
```
GET {{base_url}}/task?status=IN_PROGRESS
```

Задачи созданные админом:
```
GET {{base_url}}/task?created_by=550e8400-e29b-41d4-a716-446655440000
```

Комбинированные фильтры:
```
GET {{base_url}}/task?worker_id=660e8400-e29b-41d4-a716-446655440001&status=COMPLETED&created_by=550e8400-e29b-41d4-a716-446655440000
```

**Response (200):**
```json
{
    "tasks": [
        {
            "id": "770e8400-e29b-41d4-a716-446655440002",
            "message": "Выполнить анализ данных за Q1 2025",
            "status": "IN_PROGRESS",
            "worker_id": "660e8400-e29b-41d4-a716-446655440001",
            "created_by": "550e8400-e29b-41d4-a716-446655440000",
            "created_at": 1704067200,
            "updated_at": 1704067200
        }
    ]
}
```

---

## 📄 Document Endpoints

### 1. Add Document - JSON (Base64)
**POST** `{{base_url}}/document`  
**Headers:** 
- `Authorization: Bearer {{employee_token}}`
- `Content-Type: application/json`

```json
{
    "task_id": "770e8400-e29b-41d4-a716-446655440002",
    "filename": "report.pdf",
    "content_type": "application/pdf",
    "file_base64": "JVBERi0xLjQKJdPr6eEKMSAwIG9iago8PAovVHlwZSAvQ2F0YWxvZwovUGFnZXMgMiAwIFIKPj4KZW5kb2JqCjIgMCBvYmoKPDwKL1R5cGUgL1BhZ2VzCi9LaWRzIFszIDAgUl0KL0NvdW50IDEKL01lZGlhQm94IFswIDAgNjEyIDc5Ml0KPj4KZW5kb2JqCjMgMCBvYmoKPDwKL1R5cGUgL1BhZ2UKL1BhcmVudCAyIDAgUgovUmVzb3VyY2VzIDw8Ci9Gb250IDw8Ci9GMSA0IDAgUgo+Pgo+PgovQ29udGVudHMgNSAwIFIKPj4KZW5kb2JqCjQgMCBvYmoKPDwKL1R5cGUgL0ZvbnQKL1N1YnR5cGUgL1R5cGUxCi9CYXNlRm9udCAvSGVsdmV0aWNhCj4+CmVuZG9iago1IDAgb2JqCjw8Ci9MZW5ndGggNDQKPj4Kc3RyZWFtCkJUCi9GMSAxMiBUZgoxMDAgNzAwIFRkCihUZXN0IFBERikgVGoKRVQKZW5kc3RyZWFtCmVuZG9iagp4cmVmCjAgNgowMDAwMDAwMDAwIDY1NTM1IGYgCjAwMDAwMDAwMDkgMDAwMDAgbiAKMDAwMDAwMDA1OCAwMDAwMCBuIAowMDAwMDAwMTE1IDAwMDAwIG4gCjAwMDAwMDAyNjEgMDAwMDAgbiAKMDAwMDAwMDMxOCAwMDAwMCBuIAp0cmFpbGVyCjw8Ci9TaXplIDYKL1Jvb3QgMSAwIFIKPj4Kc3RhcnR4cmVmCjQwNQolJUVPRgo=",
    "tags": ["report", "q1-2025", "analysis"]
}
```

**Поддерживаемые типы файлов:**
- **Документы:** `application/pdf`, `application/msword`, `application/vnd.openxmlformats-officedocument.wordprocessingml.document`, `text/plain`, `application/vnd.ms-excel`, `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- **Изображения:** `image/png`, `image/jpeg`

**Response (201):**
```json
{
    "id": "507f1f77bcf86cd799439011",
    "filename": "report.pdf",
    "content_type": "application/pdf",
    "size": 1234,
    "task_id": "770e8400-e29b-41d4-a716-446655440002",
    "owner_id": "660e8400-e29b-41d4-a716-446655440001",
    "tags": ["report", "q1-2025", "analysis"],
    "uploaded_at": 1704067200
}
```

---

### 2. Add Document - Image JPEG
**POST** `{{base_url}}/document`  
**Headers:** 
- `Authorization: Bearer {{employee_token}}`
- `Content-Type: application/json`

```json
{
    "task_id": "770e8400-e29b-41d4-a716-446655440002",
    "filename": "photo.jpg",
    "content_type": "image/jpeg",
    "file_base64": "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/2wBDAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwA/8A",
    "tags": ["photo", "evidence"]
}
```

---

### 3. Add Document - Multipart Form Data
**POST** `{{base_url}}/document`  
**Headers:** `Authorization: Bearer {{employee_token}}`  
**Content-Type:** `multipart/form-data`

**Form Data:**
- `task_id`: `770e8400-e29b-41d4-a716-446655440002`
- `filename`: `screenshot.png`
- `content_type`: `image/png`
- `tags`: `screenshot,proof` (через запятую)
- `file`: [выберите файл]

---

### 4. Get Document
**GET** `{{base_url}}/document/{{document_id}}`  
**Headers:** `Authorization: Bearer {{employee_token}}`

**Response (200):**
```json
{
    "id": "507f1f77bcf86cd799439011",
    "filename": "report.pdf",
    "content_type": "application/pdf",
    "size": 1234,
    "task_id": "770e8400-e29b-41d4-a716-446655440002",
    "owner_id": "660e8400-e29b-41d4-a716-446655440001",
    "tags": ["report", "q1-2025", "analysis"],
    "uploaded_at": 1704067200,
    "file_base64": "JVBERi0xLjQKJdPr6eEKMSAwIG9iago8PAovVHlwZSAvQ2F0YWxvZwovUGFnZXMgMiAwIFIKPj4KZW5kb2JqCjIgMCBvYmoKPDwKL1R5cGUgL1BhZ2VzCi9LaWRzIFszIDAgUl0KL0NvdW50IDEKL01lZGlhQm94IFswIDAgNjEyIDc5Ml0KPj4KZW5kb2JqCjMgMCBvYmoKPDwKL1R5cGUgL1BhZ2UKL1BhcmVudCAyIDAgUgovUmVzb3VyY2VzIDw8Ci9Gb250IDw8Ci9GMSA0IDAgUgo+Pgo+PgovQ29udGVudHMgNSAwIFIKPj4KZW5kb2JqCjQgMCBvYmoKPDwKL1R5cGUgL0ZvbnQKL1N1YnR5cGUgL1R5cGUxCi9CYXNlRm9udCAvSGVsdmV0aWNhCj4+CmVuZG9iago1IDAgb2JqCjw8Ci9MZW5ndGggNDQKPj4Kc3RyZWFtCkJUCi9GMSAxMiBUZgoxMDAgNzAwIFRkCihUZXN0IFBERikgVGoKRVQKZW5kc3RyZWFtCmVuZG9iagp4cmVmCjAgNgowMDAwMDAwMDAwIDY1NTM1IGYgCjAwMDAwMDAwMDkgMDAwMDAgbiAKMDAwMDAwMDA1OCAwMDAwMCBuIAowMDAwMDAwMTE1IDAwMDAwIG4gCjAwMDAwMDAyNjEgMDAwMDAgbiAKMDAwMDAwMDMxOCAwMDAwMCBuIAp0cmFpbGVyCjw8Ci9TaXplIDYKL1Jvb3QgMSAwIFIKPj4Kc3RhcnR4cmVmCjQwNQolJUVPRgo="
}
```

---

### 5. Get Documents by Task
**GET** `{{base_url}}/document/task/{{task_id}}`  
**Headers:** `Authorization: Bearer {{admin_token}}`  
**Примечание:** Только для админов

**Response (200):**
```json
{
    "documents": [
        {
            "id": "507f1f77bcf86cd799439011",
            "filename": "report.pdf",
            "content_type": "application/pdf",
            "size": 1234,
            "task_id": "770e8400-e29b-41d4-a716-446655440002",
            "owner_id": "660e8400-e29b-41d4-a716-446655440001",
            "tags": ["report", "q1-2025", "analysis"],
            "uploaded_at": 1704067200
        }
    ]
}
```

---

### 6. Get Documents by Owner
**GET** `{{base_url}}/document/owner`  
**Headers:** `Authorization: Bearer {{employee_token}}`  
**Примечание:** Возвращает документы текущего пользователя (owner_id берется из JWT токена)

**Response (200):**
```json
{
    "documents": [
        {
            "id": "507f1f77bcf86cd799439011",
            "filename": "report.pdf",
            "content_type": "application/pdf",
            "size": 1234,
            "task_id": "770e8400-e29b-41d4-a716-446655440002",
            "owner_id": "660e8400-e29b-41d4-a716-446655440001",
            "tags": ["report", "q1-2025", "analysis"],
            "uploaded_at": 1704067200
        }
    ]
}
```

---

### 7. Delete Document
**DELETE** `{{base_url}}/document/{{document_id}}`  
**Headers:** `Authorization: Bearer {{employee_token}}`  
**Примечание:** Может удалить только владелец документа или админ

**Response (200):**
```json
{
    "success": true
}
```

---

## ⚠️ Ошибки

### 400 Bad Request
```json
{
    "error": "message is required"
}
```

### 401 Unauthorized
```json
{
    "error": "authorization header missing"
}
```

### 404 Not Found
```json
{
    "error": "task not found"
}
```

### 413 Request Entity Too Large
```json
{
    "error": "file too large"
}
```

---

## 📝 Примечания

1. **Base URL:** `http://localhost:8084` (по умолчанию)
2. **Максимальный размер файла:** 10 МБ (10,485,760 байт)
3. **JWT токен:** Действителен 3600 секунд (1 час)
4. **Статусы задач:** `IN_PROGRESS`, `COMPLETED`, `NEEDS_HELP`
5. **Роли:** `admin`, `employee`
6. **Поддерживаемые форматы файлов:**
   - PDF: `.pdf`
   - Word: `.doc`, `.docx`
   - Excel: `.xls`, `.xlsx`
   - Текст: `.txt`
   - Изображения: `.png`, `.jpg`, `.jpeg`

