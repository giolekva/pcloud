# Frontend Tech Stack
This document describes guidelines to follow when choosing a tech stack for new frontend service.

## Background
Most of the services deployed on PCloud, be it core infrastructure service or a regular application, will need frontend so users can interact with it.

## Goals
* Establish set of considerations to think about when choosing a tech stack to implement new frontend service

## Non-goals
* Determine one tech stack to be used in all frontend services

## Technichal overview
Core PCloud infrastructure consists of number of services responsible for: 
1. Authentication and authorization of the users
2. User management: creation, deletion, password reset, configuring MFA (Multi Factor Authentication)
3. Application manager facilitating installation and upgrade process of third-party applications
4. Personilized homepage giving users home screen with application launcher and personilized widgets. See [iGoogle](https://www.google.com/search?q=igoogle) for examples.

All of these services have different complexity and security threat levels. One solution will not be suitable for all of them. Whenever possible, simpler solution must be chosen. Especially services with high security threat must limit their dependencies to the minimum, to make auditing of the service as seamless as possible. Nowdays a lot can be achieved by using server-side rendered pure HTML5 and modern CSS. And that must be the first go-to approach to consider when implementing a new service. For services with traditional GUI (Graphical User Interface) application like behaviour more complex SPA (Single Page Application) solutions can be considered.

Above mentioned services are listed below in increasing order of their rough complexity:
1. Authentication service will implement login, logout and consent flows.
2. User management service will provide basic CRUD (Create, Read, Update, Delete) operations over user and group entities.
3. Application manager is a bit more complex, must provide search functionality over local cache and dynamically render form to let the administrator configuration application settings before installing it.
4. Personalized homepage will be very dynamic and will have to be implemented as an SPA.

These services can be split into two categories: 
1. Administrator facing: stability of the UI and security of such applications is most important. Administrators should not have to learn new UI with every release of the application. Such applications can be considered boring to work on, they are developed once and must require minimal changes for maintenance.
2. End user facing: usability, responsiveness and overall UX is more important here. Such applications will have high maintenance cost, as new features will have to be frequently added and UX will have to be kept up to date to modern standards and user expectations.

Server-side rendered HTML5 and CSS ([Pico.css](https://picocss.com)) with little to no Javascript will be sufficient for first category of applications. Authentication and User Management services belong to that category. Application Manager can also be put into the first category. While it may benefit from SPA like behaviour, downsides of extra dependency on NodeJS/NPM like ecosystem and or on any specific web framework outweigh the benefits they bring. Personalized homepage on the other hand must work as an SPA, and it will benefit from adopting expansive set of libraries from NPM ecosystem.

There is always new cool kid/framework (React, Vue, Svelte, ...) everyone is using in JS ecosystem to build frontend applications. On the other hand [Lit](https://lit.dev) is very close to the web standards and if needed can be utilised with above mentioned frameworks. We should consider Lit as a way to implement core PCloud components.
