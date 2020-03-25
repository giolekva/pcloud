# PFS - PCloud File System
## Overview
PFS is a core building block of PCloud providing distributed blob storage, with replication for redundancy and high availability.
It is designed along the lines of GFS (Google File System): <https://static.googleusercontent.com/media/research.google.com/en//archive/gfs-sosp2003.pdf>
PFS consists of two main components: Controller and Chunk servers. Chunk servers store actual data while controller maintains a global view and acts on changes in the cluster.

## Goals & Requirements
* It must be easy to add new chunk servers. Controller should automatically pick it up and use it for storage.
* Taking out a chunk server must trigger re-replication of chunks stored there.
* Controller must assigne chunks so that load is equally distributed among chunk servers and improve throughput.
* Blob sizes are known at the time of creation. This simplifies a design but can be reconsidered.

## Concepts used in the document:
* Blob: represents single file. Blobs have globally unique ids.
* Chunk: blobs are split into one or more chunks with equal sizes. Last chunk might be smaller than others. Chunks have globally unique ids.
* Chunk server: RPC server storing chunks.
* Controller: RPC server coordinating blob/chunk creation and their assignments to chunk servers.
* Chunk replica: same chunk might be stored on multiple chunk servers to achieve high availability. Such copies are called chunk replicas.
* Chunk assignment: list of chunk servers storing particular chunk.
* Primary replica: when uploading new chunk, one of the replicas will act as primary. Receiving data from the client.
* Secondary replica: all non-primary replicas are secondary. They replicate data from primary replica.

## Detailed design
Chunk servers maintain list of chunks they store. Actual chunk payloads will be stored on local disk using OS provided file system. Whole metadata, chunk server needs to maintain its state, must be periodically persisted on disk so chunk server can quickly recover upon failure.

Chunk ids will be represented as [RFC 4122](https://tools.ietf.org/html/rfc4122) compliant 128 bit UUID.
Chunk metadata will consist of:
```golang
type ChunkInfo struct {
        // Status of the chunk: NEW, CREATED, ..., READY
	Status    ChunkStatus
	// Total size of chunk in bytes
	Size      int
	// Number of bytes committed to disk
	Committed int
	// TODO(lekva): add file path and update bytes bellow
}
```
Total of 16 + 3 * 32 = 112 bytes are needed to store single chunk metadata. On top if this thread-safe hash map backed ChankInfoStore structure will be built with two Load and Store methods. Store method will update in memory hash map and also append it to transaction logs. Background process will compact transaction logs periodically and persist full hash map contents on disk.

Controller will not persist any data locally. Instead it will receive state of chunk servers periodically using heart beats. This makes it is easier to keep metadata stored in controller and chunk servers consistent.

Chunk server will provide RPC interface with ListChunks, CreateChunk, GetChunkInfo, ReadChunk, WriteChunk, RemoveChunk methods. ReadChunk and WriteChunk methods will ask for offset which in conjunction with ChunkInfo.Commited variable will be used to correctly access chunk payload.

Controller will provide RPC interface with CreateBlob, GetBlobInfo and RemoveBlob methods. When creating blob, controller will decide how to cut it into smaller chunks and their assignment to chunk servers, then call CreateChunk method on all involved assigned chunk servers, and return BlobInfo back to the client with blob and chunk ids. For each chunk one of the chunk servers will be chosen as a primary which will receive data from the client while all others, secondary replicas, will start a background process of replicating data from primary.

Controller will run background process assesing health of chunk servers. It will trigger new chunk creation and replication as soon as one of the chunk servers goes down. Similarly there will be a background garbage collector running whuch will trigger removal of unneccessary chunk replicas. This can happen once previously offline chunk server, whos contents have already been replciated, comes back online. Same functionallity will be used to lazily delete blobs which have been marked for deletion.