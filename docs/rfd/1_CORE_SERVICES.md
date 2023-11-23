STATUS: draft

# PCloud Core Frontend Services
This document describes frontend services needed to manage PCloud core functionalities, such as: users with groups they belong to and roles, network infrastructure, application store/manager and home/launcher application.

## Background
PCloud out of the box comes with capabilities to manage users, give them access to only specific applications installed on the platform, store so administrators can install applications from various repositories onto the platform, and launcher so platform users can easily find and use installed applications.

## Goals
1. Key concepts of the PCloud core infrastructure
2. Briefly describe high level functionalities of core services
3. Describe how different services will interact with each other

## Non-goals
Goal of this document is not to provide implementation details. Technichal implementation details for each service must be decided separately. This document merely draws the big picture to act as a starting point for further discussions.

## Technichal overview
Each PCloud envrionment consists of two virtual networks: public and private. Applications installed on public network are reachable globally, while applications installed on private network require user to be logged into the PCloud VPN. Reachability does not guarantee that user will be able to use the application, applications are free to restrict access to specific users or groups of users.

To achieve such security, PCloud platform must implement following concepts (each of which will be described in detail later in the document):
1. Registration, authentication and authorization of the users
2. Group memberships
  1. Users can be assigned to any number of groups
  2. Groups can belong to any number of other groups (inheritance)
3. Applications can be configured to restrict access to only specific set of groups
4. Users must be able to explicitly request to be added to a specific group
5. Administrator at any time must be allowed to grant or take away group membership from the user
6. (Future Work) Notion of group owners must be introduced:
  1. Owner must be able to accept/reject group membership requests
  2. Owner must be able to request group they manage to become part of the other group
7. When installing application administrator must be able to:
  1. Choose network (public/private) from which application can be reached
  2. Grant access to only specific groups
8. Set of groups with permission to access specific application can be changed at any time after application is installed

Functinoalities described above must be split into number of services, each of which is briefly described below.

### User/group Manager
1. Registration, authentication and authorization of the users
2. Creation/deletion of the groups
3. Assigning users to the group
4. Assigning groups to the outher groups
5. OAuth2 based authentication and authorization flows
  1. Registration flow
  2. Login flow
  3. Consent flow - giving access to specific details of the user (username, email, ...) to the application

This is partially implemented with basic UI. Current implementation can be found at `core/auth/ui`. It uses **Ory Kratos** to store user identities and **Ory Hydra** to implement OAuth2 flows. Notion of groups is not currently implemented.

User schema must be extended with:
1. SSH public keys - so that application using SSH based authentication can distinguish users.
2. PGP public/private key - so that communication can be encrypted when necessary. For example email service can automatically encrypt outgoing messages with user specific PGP private key.

### Application Manager
Application Manager itself does not host application configurations developed by PCloud or any other third-party provider, that is the job of the Application Repository. 
Application Repository is an external HTTP service listing (in YAML/JSON format) set of published application configurations which can be installed onto the platform. For example third-party developer can host their own Application Repository or use already existing one to publish their application. PCloud must develop first-party repository to help kickstart the platform, but over time number of third-party repositories must emerge to create healty and competitive environment for application developers.

What application configuration looks like is described in a separate document: **TBD**

Application discovery flow must look like: 
1. Administrator can make any number of Application Repositories discoverable in the manager by registering their HTTP endpoints
2. Application Manager must periodically crawl configured repositories and cache application configurations locally
3. Administrator can force manager to refresh local cache at any time

Application manager must let administrator: 
1. Browse and find applications cached locally
2. Install and configure them onto the platform
  1. Choose network
  2. Choose set of groups
  3. Choose subdomain (if necessary) using which application must be accessible
  4. Configure settings explicitly required by the application
3. Upgrade application to a newer version
4. Uninstall application
5. Install multiple instances of the same application onto the platform in isolation
6. Reconfigure already installed application
  1. Change subdomain
  2. Change application settings
  3. Change set of groups granted access

Initial implementations can be found at: 
1. Application Manager - `core/installer/welcome/appmanager.go`
2. Application Repository - `apps/app-repository`

### Securing Applications
Current implementation secures private network using [Headscale](https://github.com/juanfont/headscale) which is an OSS (open-source software) implementation of [Tailscale](https://tailscale.com). Headscale has it's own notion of users and groups, and can make any shared service accessible to specific groups. So changes in group membersip must be automatically propagated from User Manager to Headscale.

On top of Headscale based security, applications implementing OIDC (OpenID Connect) protocol can tap into User Manager service to gather information regarding currently logged in user and their groups, and make decisions accordingly. Applications which do not implement OIDC themselves, must have [OAuth2 Proxy](https://github.com/oauth2-proxy/oauth2-proxy) running in front of them.
