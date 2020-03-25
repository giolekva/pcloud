# PCloud File System
## Overview
PFS is a core building block of PCloud providing distributed blob storage, with replication for redundancy and high availability.
It is designed along the lines of GFS (Google File System): <https://static.googleusercontent.com/media/research.google.com/en//archive/gfs-sosp2003.pdf>
PFS consists of two main components: Controller and Chunk servers. Chunk servers store actual data while controller maintains a global view and acts on changes in the cluster.

## Goals
* It must be easy to add new chunk servers. Controller should automatically pick it up and use it for storage.
* Taking out a chunk server must trigger re-replication of chunks stored there.
* Controller must assigne chunks so that load is equally distributed among chunk servers and improve throughput.

# Detailed design
