# FlashORM v2.0.8 Release Notes

## 🔧 Bug Fixes & Enhancements

### MySQL ENUM & Foreign Key Support

**ENUM Type Support:**
- ✅ **Go Generator** - Properly generates ENUM type definitions with constants for MySQL inline ENUMs
- ✅ **TypeScript Generator** - Converts `enum('a','b','c')` to union types `'a' | 'b' | 'c'`
- ✅ **Nullable ENUMs** - Uses `sql.NullString` for nullable ENUM fields in Go (no more undefined `sql.Null<CustomType>` errors)
- ✅ **ENUM Default Values** - Automatically adds quotes around ENUM default values when pulling schema

**Foreign Key Relationships:**
- ✅ **Studio Visualization** - MySQL foreign key relationships now display correctly in schema diagram
- ✅ **Clickable FK Cells** - Table cells with foreign keys are now clickable in Studio data browser
- ✅ **GetTableColumns** - Extracts `ForeignKeyTable`, `ForeignKeyColumn`, and `OnDeleteAction` from MySQL information_schema
- ✅ **GetAllTablesColumns** - Batch query optimization now includes FK relationship data

**MySQL Connection Improvements:**
- ✅ **URL Format Support** - Auto-converts `mysql://user:pass@host:port/db` to DSN format
- ✅ **SSL Parameters** - Automatically translates PostgreSQL-style SSL parameters to MySQL driver format
  - `ssl-mode=REQUIRED` → `tls=skip-verify`
  - `ssl-mode=DISABLED` → `tls=false`
  - `sslmode=require` → `tls=skip-verify`

**Usage Example:**
```go
// Generated ENUM types (Go)
type UsersUserType string
const (
    UsersUserTypeCustomer UsersUserType = "customer"
    UsersUserTypeStylist  UsersUserType = "stylist"
)

type Users struct {
    UserType      UsersUserType   `json:"user_type" db:"user_type"`
    AccountStatus sql.NullString  `json:"account_status" db:"account_status"` // nullable ENUM
}
```

```typescript
// Generated TypeScript types
interface Users {
  user_type: 'customer' | 'stylist' | 'nailist' | 'eyelist';
  account_status: 'active' | 'suspended' | 'deactivated' | null;
}
```

## 📦 Installation

### NPM
```bash
npm install -g flashorm
```

### Go
```bash
go install github.com/Lumos-Labs-HQ/flash@latest
```

### Binary Download
Download from [GitHub Releases](https://github.com/Lumos-Labs-HQ/flash/releases)

## 📚 Documentation

- [Main Documentation](https://github.com/Lumos-Labs-HQ/flash)
- [NPM Package](https://www.npmjs.com/package/flashorm)
- [TypeScript Examples](https://github.com/Lumos-Labs-HQ/flash/tree/main/example/ts)
- [Technology Stack](https://github.com/Lumos-Labs-HQ/flash/blob/main/docs/TECHNOLOGY_STACK.md)

## 💬 Feedback

- 🐛 [Report bugs](https://github.com/Lumos-Labs-HQ/flash/issues)
- 💡 [Request features](https://github.com/Lumos-Labs-HQ/flash/issues)
- ⭐ [Star the repo](https://github.com/Lumos-Labs-HQ/flash)

---

**Download:** [v2.0.8 Release](https://github.com/Lumos-Labs-HQ/flash/releases/tag/v2.0.8)

**NPM:** `npm install -g flashorm`

**Go:** `go install github.com/Lumos-Labs-HQ/flash@latest`