# PCloud
PCloud is a set of distributed infrustructure tools meant for setting up a personal cloud on commodity hardware such as Raspberry Pi.
Goal of the project is to provide:
* Object and file storage: distributed storage with configurable replication for durability and redundancy
* Knowledge graph: storing graph shaped data representing user/application generated content and relations between them
* Application infrastructure: to easily set up and run third-party applications, where applications can communicate with each other and produce/consume knowledge graph nodes
* Search infrastructure: indexing knowledge graph and making it searchable

To prove viability of the project first milestone will be to build fully functional image storage and indexing infrustructure. User experience setting it up will look something like:
* Set up PCloud on 3 or more servers and pair mobile device with it
* Configure IFTTT (if this then that) like worklfow to automatically:
  * Back up every newly taken picture on PCloud
  * Run face detection app on backed up pictures and store this information in Metadata service
  * Index face annotations and make them searchable

User must be able to configure all of these from previously paired mobile device.

# Status
Three core infrastructure services have been prototyped:
* Knowledge Graph API: GraphQL based api with extensible schema
  * Provides CRUD operations
  * Auto-generates appropriate events upon data modification and includes them within mutation transaction
  * Applicatioin installed by Application Manager (see below) can extend Knowledge Graph schema
* Application Manager: supports installing third-party applications by uploading configartion archive via web ui.
  * Application configuration consists of:
    * Schema extension (optional): if provided Knowledge Graph schema will be extended with new types and relations.
    * Actions (optional): application can define any number of actions which can be invoked from other applications. Actions are parametrized.
    * Initialization action (optional): application can configure action, provided possibly by other application, to be run post installation.
    * Triggers (optional): applications can set up triggers on Knowledge Graph mutations. Triggers run actions.
  * Applications are instulled into separate namespaces for isolation.
* Event Processor: monitors changes in Knowledge Graph and triggers actions registered by applications installed using Application Manager.
  * It is basically a state machine moving events from NEW to IN_PROGRESS to DONE states.

On top of this we are running five "third-party" applications:
* Random Puppy:
  * Does not use any PCloud features
  * Deploys web server with ingress
* Object Store:
  * Provides AWS S3 compatible API
  * Exposes create-bucket-with-webhook action so other applications can create buckets and receive notifications when new objects are created.
  * Important detail here is that object store itself is installed as a third-party app. This means other storage solution can be integrated with PCloud infrustructure without changing PCloud itself.
* Image importer:
  * Registers new Knowledge Graph node type: Image
  * Using Object Store create-bucket-with-webhook action to create new images bucket and register itself as a listener
  * For every new object creates new Image node in Knowledge Graph
* Face Detector:
  * Registers new Knowledge Graph node tupe ImageSegment and extends previously created Image type with their relation.
  * Registers trigger on new Image nodes with action running face detection algorithm, which upon completion creates ImageSegment node for each face and attaches them to source Image.
* Photos UI:
  * Web based photo gallery application consuming Image and ImageSegment nodes via Knowledge Graph API.

Note that since Object Store is standalone application with it's own API, new third-party applications can be developed to import data from cloud providers, such as Google Drive, where users might currently be storing their personal data.

# Next steps:
* To further isolate applications from each other, namespaces they are created must be configured so they can communicate only with core PCloud services. PCloud API will take responisibility of forwarding action requests to applications.
* Once previous step is implemented, PCloud API can enforce ACLs and make application to application communication transparent to the PCloud owner. Application during instalation will list actions, provided by other applications, it wants to use. And all of these can be reviewed and possibly rejected by the owner.
* Make Knowledge Graph types namespaced. Right now applications add new types into global namespace which can cause conflicts. To avoid this Knowledge Graph API can build namespaces on top of Dgraph GraphQL schemas.
* Support replaying events so applications can "react" to events created before installation. This way Face Detection app will be able to process images created before face detector was installed.
* Make Tensorflow Facenet model work on Raspberry Pi.