---
title: MongoDB Guide
description: Using Flash ORM with MongoDB databases
---

# MongoDB Guide

Complete guide to using Flash ORM with MongoDB databases, including document modeling, aggregation pipelines, and MongoDB-specific features.

## Table of Contents

- [Installation & Setup](#installation--setup)
- [MongoDB-Specific Features](#mongodb-specific-features)
- [Document Modeling](#document-modeling)
- [Query Operations](#query-operations)
- [Aggregation Pipelines](#aggregation-pipelines)
- [Indexing Strategies](#indexing-strategies)
- [GridFS for Files](#gridfs-for-files)
- [Change Streams](#change-streams)
- [Transactions](#transactions)
- [Performance Optimization](#performance-optimization)

## Installation & Setup

### MongoDB Installation

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install mongodb

# macOS with Homebrew
brew install mongodb-community
brew services start mongodb-community

# Docker
docker run --name mongodb -p 27017:27017 -d mongo:6.0
```

### Connection Configuration

```bash
# Basic connection
export DATABASE_URL="mongodb://localhost:27017/myapp"

# With authentication
export DATABASE_URL="mongodb://username:password@localhost:27017/myapp"

# Replica set
export DATABASE_URL="mongodb://localhost:27017,localhost:27018,localhost:27019/myapp?replicaSet=rs0"

# Atlas connection
export DATABASE_URL="mongodb+srv://username:password@cluster.mongodb.net/myapp"
```

### Flash ORM Setup

```bash
# Initialize with MongoDB
flash init --mongodb

# Verify connection
flash status
```

## MongoDB-Specific Features

### Connection Parameters

```env
# Connection string options
DATABASE_URL=mongodb://localhost:27017/myapp?maxPoolSize=10&minPoolSize=5&maxIdleTimeMS=30000

# SSL/TLS
DATABASE_URL=mongodb://localhost:27017/myapp?ssl=true&tlsCertificateKeyFile=/path/to/cert.pem

# Authentication
DATABASE_URL=mongodb://user:pass@localhost:27017/myapp?authSource=admin

# Read preferences
DATABASE_URL=mongodb://localhost:27017/myapp?readPreference=secondaryPreferred
```

### MongoDB-Specific Config

```json
// flash.config.json
{
  "database": {
    "provider": "mongodb",
    "url_env": "DATABASE_URL",
    "mongodb": {
      "database": "myapp",
      "max_pool_size": 10,
      "min_pool_size": 5,
      "max_idle_time_ms": 30000,
      "read_preference": "secondaryPreferred",
      "write_concern": "majority"
    }
  }
}
```

## Document Modeling

### Schema Definition

```javascript
// MongoDB collections (defined in schema files)
// db/schema/users.js
{
  "collection": "users",
  "indexes": [
    { "key": { "email": 1 }, "unique": true },
    { "key": { "username": 1 }, "unique": true }
  ],
  "validation": {
    "validator": {
      "$jsonSchema": {
        "bsonType": "object",
        "required": ["email", "username"],
        "properties": {
          "email": { "bsonType": "string", "pattern": "^.+@.+$" },
          "username": { "bsonType": "string", "minLength": 3 },
          "isActive": { "bsonType": "bool", "default": true },
          "createdAt": { "bsonType": "date" }
        }
      }
    }
  }
}
```

### Document Structure

```javascript
// User document example
{
  "_id": ObjectId("507f1f77bcf86cd799439011"),
  "email": "john@example.com",
  "username": "johndoe",
  "profile": {
    "firstName": "John",
    "lastName": "Doe",
    "avatar": "https://example.com/avatar.jpg"
  },
  "preferences": {
    "theme": "dark",
    "notifications": {
      "email": true,
      "push": false
    }
  },
  "posts": [
    ObjectId("507f1f77bcf86cd799439012"),
    ObjectId("507f1f77bcf86cd799439013")
  ],
  "isActive": true,
  "createdAt": ISODate("2024-01-15T10:30:00Z"),
  "updatedAt": ISODate("2024-01-15T10:30:00Z")
}
```

### Schema Validation

```javascript
// Collection validation rules
db.runCommand({
  "collMod": "users",
  "validator": {
    "$jsonSchema": {
      "bsonType": "object",
      "required": ["email", "username", "createdAt"],
      "properties": {
        "email": {
          "bsonType": "string",
          "pattern": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
        },
        "username": {
          "bsonType": "string",
          "minLength": 3,
          "maxLength": 30
        },
        "isActive": {
          "bsonType": "bool"
        },
        "createdAt": {
          "bsonType": "date"
        }
      }
    }
  },
  "validationLevel": "moderate",
  "validationAction": "warn"
});
```

## Query Operations

### Basic CRUD Operations

```javascript
// db/queries/users.js

// Find user by ID
db.users.findOne({ _id: ObjectId("507f1f77bcf86cd799439011") });

// Find users by criteria
db.users.find({ isActive: true });

// Insert user
db.users.insertOne({
  email: "jane@example.com",
  username: "janedoe",
  isActive: true,
  createdAt: new Date()
});

// Update user
db.users.updateOne(
  { _id: ObjectId("507f1f77bcf86cd799439011") },
  {
    $set: { "profile.firstName": "Jane" },
    $currentDate: { updatedAt: true }
  }
);

// Delete user
db.users.deleteOne({ _id: ObjectId("507f1f77bcf86cd799439011") });
```

### Advanced Queries

```javascript
// Complex queries
// Find users with posts in specific categories
db.users.find({
  posts: {
    $in: db.posts.find({
      category: "technology",
      published: true
    }).map(p => p._id)
  }
});

// Text search
db.posts.find({
  $text: { $search: "mongodb aggregation" }
});

// Geospatial queries
db.locations.find({
  location: {
    $near: {
      $geometry: { type: "Point", coordinates: [-122.4194, 37.7749] },
      $maxDistance: 1000
    }
  }
});
```

### Query Files

```javascript
// db/queries/users.js

// name: GetUserByID :one
db.users.findOne({ _id: ObjectId($1) });

// name: GetUsersByStatus :many
db.users.find({ isActive: $1 }).sort({ createdAt: -1 });

// name: CreateUser :one
db.users.insertOne({
  email: $1,
  username: $2,
  isActive: true,
  createdAt: new Date(),
  updatedAt: new Date()
});

// name: UpdateUser :exec
db.users.updateOne(
  { _id: ObjectId($1) },
  {
    $set: {
      email: $2,
      username: $3,
      updatedAt: new Date()
    }
  }
);

// name: SearchUsers :many
db.users.find({
  $or: [
    { username: { $regex: $1, $options: 'i' } },
    { email: { $regex: $1, $options: 'i' } }
  ],
  isActive: true
}).limit($2).skip($3);
```

## Aggregation Pipelines

### Basic Aggregations

```javascript
// Count posts by user
db.posts.aggregate([
  {
    $group: {
      _id: "$userId",
      postCount: { $sum: 1 },
      lastPostDate: { $max: "$createdAt" }
    }
  },
  {
    $lookup: {
      from: "users",
      localField: "_id",
      foreignField: "_id",
      as: "user"
    }
  },
  {
    $unwind: "$user"
  },
  {
    $project: {
      username: "$user.username",
      postCount: 1,
      lastPostDate: 1
    }
  }
]);
```

### Complex Pipelines

```javascript
// User engagement analytics
db.users.aggregate([
  // Stage 1: Lookup user posts
  {
    $lookup: {
      from: "posts",
      localField: "_id",
      foreignField: "userId",
      as: "posts"
    }
  },

  // Stage 2: Lookup user comments
  {
    $lookup: {
      from: "comments",
      localField: "_id",
      foreignField: "userId",
      as: "comments"
    }
  },

  // Stage 3: Calculate engagement metrics
  {
    $project: {
      username: 1,
      email: 1,
      postCount: { $size: "$posts" },
      commentCount: { $size: "$comments" },
      engagementScore: {
        $add: [
          { $multiply: [{ $size: "$posts" }, 2] },
          { $size: "$comments" }
        ]
      },
      lastActivity: {
        $max: [
          { $max: "$posts.createdAt" },
          { $max: "$comments.createdAt" }
        ]
      }
    }
  },

  // Stage 4: Filter active users
  {
    $match: {
      engagementScore: { $gte: 5 },
      lastActivity: { $gte: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000) }
    }
  },

  // Stage 5: Sort by engagement
  {
    $sort: { engagementScore: -1 }
  }
]);
```

### Aggregation Queries

```javascript
// db/queries/analytics.js

// name: GetUserEngagementStats :many
db.users.aggregate([
  {
    $lookup: {
      from: "posts",
      localField: "_id",
      foreignField: "userId",
      as: "posts"
    }
  },
  {
    $lookup: {
      from: "comments",
      localField: "_id",
      foreignField: "userId",
      as: "comments"
    }
  },
  {
    $project: {
      username: 1,
      postCount: { $size: "$posts" },
      commentCount: { $size: "$comments" },
      engagementScore: {
        $add: [
          { $multiply: [{ $size: "$posts" }, 2] },
          { $size: "$comments" }
        ]
      }
    }
  },
  { $sort: { engagementScore: -1 } },
  { $limit: $1 }
]);

// name: GetMonthlyPostStats :one
db.posts.aggregate([
  {
    $match: {
      createdAt: {
        $gte: new Date($1),
        $lt: new Date($2)
      }
    }
  },
  {
    $group: {
      _id: {
        year: { $year: "$createdAt" },
        month: { $month: "$createdAt" }
      },
      postCount: { $sum: 1 },
      uniqueAuthors: { $addToSet: "$userId" }
    }
  },
  {
    $project: {
      month: "$_id",
      postCount: 1,
      uniqueAuthorsCount: { $size: "$uniqueAuthors" }
    }
  }
]);
```

## Indexing Strategies

### Index Types

```javascript
// Single field index
db.users.createIndex({ email: 1 });

// Compound index
db.posts.createIndex({ userId: 1, createdAt: -1 });

// Text index
db.posts.createIndex({ title: "text", content: "text" });

// Geospatial index
db.locations.createIndex({ coordinates: "2dsphere" });

// Hashed index
db.sessions.createIndex({ sessionId: "hashed" });

// Partial index
db.logs.createIndex(
  { level: 1, timestamp: -1 },
  { partialFilterExpression: { level: { $in: ["ERROR", "WARN"] } } }
);

// TTL index
db.sessions.createIndex(
  { expiresAt: 1 },
  { expireAfterSeconds: 0 }
);
```

### Index Management

```javascript
// List indexes
db.users.getIndexes();

// Drop index
db.users.dropIndex({ email: 1 });

// Rebuild indexes
db.users.reIndex();

// Index usage statistics
db.users.aggregate([
  { $indexStats: {} }
]);
```

### Index Optimization

```javascript
// Analyze query performance
db.posts.find({ userId: ObjectId("..."), published: true })
  .explain("executionStats");

// Create optimal indexes
db.posts.createIndex({
  userId: 1,
  published: 1,
  createdAt: -1
});

// Covered query index
db.users.createIndex({
  isActive: 1,
  createdAt: -1
}, {
  name: "active_users_covering"
});
```

## GridFS for Files

### GridFS Setup

```javascript
// Initialize GridFS bucket
const bucket = new mongodb.GridFSBucket(db, {
  bucketName: 'uploads'
});

// Upload file
const uploadStream = bucket.openUploadStream('example.jpg', {
  metadata: {
    userId: ObjectId("507f1f77bcf86cd799439011"),
    contentType: 'image/jpeg'
  }
});

fs.createReadStream('./example.jpg')
  .pipe(uploadStream)
  .on('error', (error) => console.error(error))
  .on('finish', () => console.log('File uploaded'));
```

### GridFS Operations

```javascript
// Download file
const downloadStream = bucket.openDownloadStreamByName('example.jpg');

downloadStream.pipe(fs.createWriteStream('./downloaded.jpg'))
  .on('error', (error) => console.error(error))
  .on('finish', () => console.log('File downloaded'));

// List files
const files = await bucket.find({
  'metadata.userId': ObjectId("507f1f77bcf86cd799439011")
}).toArray();

// Delete file
await bucket.delete(fileId);
```

### GridFS Queries

```javascript
// db/queries/files.js

// name: UploadFile :one
// (Handled by GridFS bucket)

// name: GetUserFiles :many
db.fs.files.find({
  'metadata.userId': ObjectId($1)
}).sort({ uploadDate: -1 });

// name: DeleteFile :exec
db.fs.files.deleteOne({ _id: ObjectId($1) });

// name: GetFileInfo :one
db.fs.files.findOne({ _id: ObjectId($1) });
```

## Change Streams

### Change Stream Setup

```javascript
// Watch for changes
const changeStream = db.collection('posts').watch();

// Listen for changes
changeStream.on('change', (change) => {
  console.log('Change detected:', change);

  switch (change.operationType) {
    case 'insert':
      console.log('New post:', change.fullDocument);
      break;
    case 'update':
      console.log('Post updated:', change.documentKey);
      break;
    case 'delete':
      console.log('Post deleted:', change.documentKey);
      break;
  }
});

// Filter changes
const filteredStream = db.collection('posts').watch([
  {
    $match: {
      'fullDocument.published': true
    }
  }
]);
```

### Change Stream Queries

```javascript
// db/queries/changes.js

// name: WatchPostChanges :stream
db.posts.watch([
  {
    $match: {
      operationType: { $in: ['insert', 'update', 'delete'] }
    }
  }
]);

// name: WatchUserActivity :stream
db.users.watch([
  {
    $match: {
      'fullDocument.isActive': true
    }
  },
  {
    $project: {
      userId: '$fullDocument._id',
      action: '$operationType',
      timestamp: '$clusterTime'
    }
  }
]);
```

## Transactions

### Multi-Document Transactions

```javascript
// Start a session
const session = client.startSession();

try {
  await session.withTransaction(async () => {
    // Transfer credits between users
    const fromUser = await db.users.findOne(
      { _id: ObjectId("507f1f77bcf86cd799439011") },
      { session }
    );

    const toUser = await db.users.findOne(
      { _id: ObjectId("507f1f77bcf86cd799439012") },
      { session }
    );

    if (fromUser.credits < 100) {
      throw new Error('Insufficient credits');
    }

    // Update both users
    await db.users.updateOne(
      { _id: fromUser._id },
      { $inc: { credits: -100 } },
      { session }
    );

    await db.users.updateOne(
      { _id: toUser._id },
      { $inc: { credits: 100 } },
      { session }
    );

    // Log transaction
    await db.transactions.insertOne({
      fromUserId: fromUser._id,
      toUserId: toUser._id,
      amount: 100,
      timestamp: new Date()
    }, { session });
  });

  console.log('Transaction completed successfully');
} catch (error) {
  console.error('Transaction failed:', error);
} finally {
  await session.endSession();
}
```

### Transaction Queries

```javascript
// db/queries/transactions.js

// name: TransferCredits :exec
// (Use multi-document transaction)

// name: GetTransactionHistory :many
db.transactions.find({
  $or: [
    { fromUserId: ObjectId($1) },
    { toUserId: ObjectId($1) }
  ]
}).sort({ timestamp: -1 }).limit($2);
```

## Performance Optimization

### Connection Optimization

```javascript
// Connection pool configuration
const client = new MongoClient(uri, {
  maxPoolSize: 10,
  minPoolSize: 5,
  maxIdleTimeMS: 30000,
  serverSelectionTimeoutMS: 5000,
  socketTimeoutMS: 45000,
  bufferMaxEntries: 0,
  bufferCommands: false
});
```

### Read Preferences

```javascript
// Read from secondary for analytics
db.collection('analytics').find(query).readPref('secondaryPreferred');

// Read from primary for critical data
db.collection('users').find(query).readPref('primary');
```

### Write Concerns

```javascript
// Strong consistency
db.collection('orders').insertOne(doc, { writeConcern: { w: 'majority' } });

// Fast writes
db.collection('logs').insertOne(doc, { writeConcern: { w: 0 } });
```

### Sharding

```javascript
// Enable sharding on database
sh.enableSharding("myapp");

// Shard collection
sh.shardCollection("myapp.posts", { userId: 1 });

// Check shard status
sh.status();
```

### Profiling

```javascript
// Enable profiling
db.setProfilingLevel(2, { slowms: 100 });

// View slow queries
db.system.profile.find().sort({ ts: -1 }).limit(5);

// Disable profiling
db.setProfilingLevel(0);
```

MongoDB's document-oriented nature and rich feature set make it ideal for applications requiring flexible schemas and complex queries. Flash ORM provides comprehensive support for MongoDB's unique capabilities while maintaining type safety and consistency across all supported languages.
