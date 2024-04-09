# DODO Homepage
Outlines the layout and functionality of the PCloud homepage, specifying the arrangement and behavior of its panels and content inside it.

## Background Information
PCloud customers will want to have all of their applications in one place. This homepage will serve as a central hub for customers, allowing them to easily switch between apps. It's has indirect relationship with AppManager, user with corresponding permission will be able to install/uninstall apps.

## Goals
- This will be a main page for users.
- Authenticated users should be able to search and run applications.
- [Stretch] Monitor applcation health.
- [Stretch] Documentation or some sort of small tutorial how to use application.

## Non-Goals
- Installing applications should not be a responsibility of the launcher, that is taken care of app manager. 
- Uninstalling applications should not be in the initial implementation of the launcher.

## Technichal overview
The homepage will serve a single page at the root path `/`, dedicated to presenting authenticated users with a list of applications they have permission to use. Users will get assigned groups via the memberships API. For implementing web server the standard [net/http](https://pkg.go.dev/net/http) is sufficient. The server will feature only one handler:
* `GET /` This handler will retrieve all the necessary details to render information of application and run it. This information includes the application's Name, Description, Documentation, Domain, Icon, Health status, and any additional relevant details. These details will be obtained from corresponding cue files of the applications.
Storage component will have following interface:
```go
type AppLauncherInfo struct {
	Name         string
	Description  string
	Icon         template.HTML
	HelpDocument []HelpDocument
	Url          string
}

type HelpDocument struct {
	Title    string
	Contents string
	Children []HelpDocument
}

type AppDirectory interface {
	GetAllAppsInfo() ([]AppLauncherInfo, error)
}
```
* Application information can be retrived from Application Manager Service.
- ### UI
Single HTML template will be sufficient to render the home page via [html/template](https://pkg.go.dev/html/template) package. And [embed](https://pkg.go.dev/embed) package will be used to embed said HTML template into the final binary.

- Divide the homepage into two independent panels, Left and Right:
- Ensure non-scrollability of the entire page; any scrolling needed should be confined within specific panels.
- Implement a minimization option for left panel.
- Display authenticated user information at the top of left panel.
   - Render user's icon/image, possible with the user's name. Enable interaction with the icon to display user info and possible editing options. This information could be rendered in a modal window.
   - Include additional small user information (if necessary).
   - When minimized, show only the user icon.
- Under User icon will be search icon and a vertical navigation bar with a list of applications inside.
  - The search icon should open a modal window with a search bar, allowing users to filter applications by name. Clicking on the search output should open the application in Right panel.
  - Display app icons with names and optional additional information.
    - Include Rendering app manager(app store) icon, and if someone wants to install new app, they can open app manager, search for app and install it.
  - If left side panel is minimized, show only app icons.
  - Implement Hover above App Icon functionality. It should show tooltip on right side of App icon. 
    - Add Help button inside tooltip. Apon clicking on it, Documentation modal of application should show up.
    - Create separate Documentation Modal for each application.
    - Documentation modal must be divided into two, left and right panels.
      - Left panel should be underodred list of Documentation titles and after clicking any title, corresponding content should be rendered in right panel.
      - Documentation can be nested. We have to use recursion to get all Documentation titles and corresponding content. This can be done directly inside an HTML file using Go Templates.
      - When DOM loads we create all documentation content tags, but they will be hidden.
      - Documentation content will be displayed only when interacting with the corresponding Title. To achieve this, we must create event listeners. To differentiate which title should open which content, we need to generate unique IDs for every tag. The relationship between the Title tag ID and the corresponding content tag ID must be maintained. Specifically, the Title ID should be `title-generatedID`, and the corresponding content ID should be `help-content-generatedID`, where the generated ID for the corresponding pair remains the same. We can generate these IDs directly inside an HTML file using the library [Sprig](https://github.com/Masterminds/sprig).
    - Documentation modal will have application name on left top position and close icon on top right corner.
  - Enable app launch only for authenticated user by clicking on the respective icon.
    - Each App Icon will have JS event listener, which will open coresponding application in Right panel in iframe tag.
    - The icon of the currently running app can be highlighted with a green circle.
