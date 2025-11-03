# Graft Studio

A Prisma Studio-like web interface for viewing and editing your database.

## Features

✅ **View Data**: Browse all tables and their data in a clean UI
✅ **Inline Editing**: Double-click any cell to edit
✅ **Batch Save**: Make multiple changes and save them all at once
✅ **Add Records**: Create new rows with a simple form
✅ **Delete Records**: Remove rows with confirmation
✅ **Pagination**: Handle large tables efficiently (50 rows per page)
✅ **Search Tables**: Quickly find tables in the sidebar
✅ **Real-time Updates**: Changes reflect immediately in the UI

## Usage

```bash
# Start Graft Studio (opens browser automatically)
graft studio

# Start on custom port
graft studio --port 3000

# Start without opening browser
graft studio --browser=false
```

## How It Works

1. **Select a Table**: Click any table in the left sidebar
2. **View Data**: Table data loads in the main grid
3. **Edit Cells**: Double-click any cell to edit
4. **Save Changes**: Click "💾 Save Changes" button to commit
5. **Add Rows**: Click "+ Add Record" to insert new data
6. **Delete Rows**: Click 🗑️ icon to remove a row

## Keyboard Shortcuts

- `Enter` - Confirm cell edit
- `Escape` - Cancel cell edit
- `Ctrl+S` - Save changes (coming soon)

## Architecture

- **Backend**: Go + Fiber (fast HTTP server)
- **Frontend**: Vanilla JS + Tailwind CSS (no build step)
- **Templates**: Go html/template
- **Database**: Uses existing Graft database adapters

## File Structure

```
web/studio/
├── templates/
│   └── index.html          # Main UI template
├── static/
│   ├── css/
│   │   └── studio.css      # Custom styles
│   └── js/
│       └── studio.js       # Frontend logic
└── README.md
```

## API Endpoints

```
GET    /                           # Main UI
GET    /api/tables                 # List all tables
GET    /api/tables/:name           # Get table data (paginated)
POST   /api/tables/:name/save      # Save changes
POST   /api/tables/:name/add       # Add new row
DELETE /api/tables/:name/rows/:id  # Delete row
```

## Performance

- **Pagination**: Loads 50 rows at a time
- **Lazy Loading**: Only fetches data when table is selected
- **Optimistic UI**: Changes appear instantly
- **Batch Updates**: All edits saved in single transaction

## Future Enhancements

- [ ] Advanced filtering and search
- [ ] Column sorting
- [ ] Foreign key navigation
- [ ] Bulk operations
- [ ] Export filtered data
- [ ] Dark mode
- [ ] Keyboard shortcuts
- [ ] Undo/redo
- [ ] Query builder UI
