# MongoDB Studio Support

Flash ORM Studio now supports MongoDB for viewing and browsing collections!

## Usage

Simply pass a MongoDB connection string to the `--db` flag:

```bash
flash studio --db "mongodb://localhost:27017/mydb"
```

Or with authentication:

```bash
flash studio --db "mongodb://username:password@localhost:27017/mydb"
```

## Features

- ✅ View all collections
- ✅ Browse documents with pagination
- ✅ Auto-detect field types from documents
- ✅ View document counts
- ✅ JSON document display

## Limitations

- Read-only mode (no editing/deleting via Studio UI)
- Schema is inferred from first 100 documents
- No migration support (MongoDB is schema-less)
- No code generation for MongoDB

## Example

```bash
# Start MongoDB locally
docker run -d -p 27017:27017 mongo:latest

# Open Studio
flash studio --db "mongodb://localhost:27017/test"
```

Studio will open at `http://localhost:5555` and show your MongoDB collections!

## Notes

- Collections appear as "tables" in the UI
- Documents appear as "rows"
- Field types are inferred: string, int, double, bool, object, array, date
- ObjectId fields are displayed as strings
