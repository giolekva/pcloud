# PCloud Vision

## Goals

PCloud aims to be a high level Distributed Operating System, abstracting away all the complexities associated with distributed systems and giving users simple and intuitive UI to install third-party applications with the goal of managing and controlling their personal data. Goal is PCloud to be able to run equivalents of any currently "freely" provided cloud services, such as Gmail, Dropbox, and others, either on-premises hardware or on rented infrastructure without compromising privacy and security of its users.

Here is how I envision it being used by Lex, who is not tech-savvy in any way, starting from initial setup:
1. Lex orders pre-build PCloud box online.
2. Once box arrives, Lex plugs ethernet cable into it and powers it on.
3. Lex downloads PCloud application on their mobile device, which is connected to the same private network as PCloud box.
4. Upon opening the mobile application, Lex is greeted with the notification that PCloud box has been detected in the network and is ready to be paired.
5. Upon pairing, PCloud creates hollow account with administrative privileges on the system, generates and associates authentication token with it and shares token with the paired mobile application.
6. That's it, initial setup is done and Lex can install applications on PCloud from their mobile device.
7. In the same paired PCloud mobile application Lex is presented with Application Marketplace where they can find applications like Email Server, Photo Gallery and others.
8. Application Marketplace will provide user reviews and ratings for each application so Lex can easily decide which ones to use.
9. Lex can install applications on PCloud with one click on "Install" button in Application Marketplace.
10. Let's say Lex installs one of the Email Servers on PCloud, to use it Lex can authenticate any email client of their choosing be it one provided by the mobile OS (Mail on iOS) or any other thirt-party one.
11. Lex installs PCloud client application on other devices, authenticates with personal credentials and enable VPN so all their devices can communicate within each other securely.

## Technical Details

PCloud box Lex ordered will come with core services and couple of first-party applications installed on it. Any of these first-party applications later can be uninstalled and replaced with third-party ones if Lex chooses to do so.

Core services are:
* API Service: provides permissioned access to Knowledge Graph and other resources stored inside individual applications.
* Application Manager: provides API to install, upgrade and uninstall third-party applications on the platform.
* Event Processor: monitors mutations in Knowledge Graph and triggers actions registered by third-party application.

First-party applications are:
* Application Marketplace: indexes and makes publicly published applications locally so users can search and install them easily via WebUI or mobile application.
* VPN Provider: enables secure virtual private networking among users devices with optional egress node. Users should be able to give access to their devices to other PCloud users from the same instance or anyone on the web.
* DNS Server: configures DNS server within the VPN and can be used configure denylist of endpoints to block advertisements. Pihole can be used here.
