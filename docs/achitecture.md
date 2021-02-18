
# PCloud Architecture 
PCloud is a bundle of several services: Knowledge Graph, AppManager, EventProcessor, VPN, OAuth.


## Knowledge Graph
Knowledge graph contains all the data and abstracts away PCloud persistance. Gives capability to store and retrieve information. It's a service that runs on a dedicated node in the Kubernetes cluster.
Let's use layered architecture for the Knowledge Graph implementation.


### Store Layer
Store Layer abstracts away the actual DB implementation. DB supported will be Postgres, but adding or changing to MySQL or any other DBMS will not affect the interface. In future different parts of the interface might store the data in the different Databases - relational, GraphDB, DocumentDB or any other.

Store layer contains different stores such as: User, Permission, OAuth, Network, Device, Password Stores. Right now all of these essentially are different tables in the Postgres.



### Application Layer
Application Layer acts as a middleware, all the permission checking is done there. AppLayer "has" the Store and performs appropriate actions. AppLayer is communicated through an interface.


### REST API Layer, gRPC Layer, CLI Layer? ...
These layers are on top of the Application Layer. Other services (AppManager) communicate to the Knowledge graph from here.


## App Manager
...

## Event Processor
...