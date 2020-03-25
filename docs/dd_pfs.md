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
	// Number of bytes committed on disk
	Committed int
}
```
