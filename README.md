# PCloud
PCloud is a set of distributed infrustructure tools meant for setting up a personal cloud on commodity hardware such as Raspberry Pi.
Goal of the project is to provide:
* Blob storage: distributed file storage with configurable replication for redundancy
* Namespace service: built on top of Blob storage to expose it, at least in read only mode, as a mountable file system
* Metadata service: storing Blob annotations such as objects and their locations detected in the pictures
* Search infrastructure: to index Blob metadata and provide search API
* App infrastructure: to easily set up and run third party applications such as custom in picture object detection models

To prove viability of the project first milestone will be to build fully functional image storage and indexing infrustructure. User experience setting it up will look somethink like:
* Set up PCloud on 3 or more servers and pair mobile device with it
* Configure IFTTT (if this then that) like worklfow to automatically:
.. * Back up every newly taken picture on PCloud
.. * Run face detection app on backed up pictures and store this information in Metadata service
.. * Index face annotations and make them searchable
User must be able to configure all of these from previously paired mobile device.