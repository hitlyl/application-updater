# README

## About

This is the official Wails Vue-TS template.

You can configure the project by editing `wails.json`. More information about the project settings can be found
here: https://wails.io/docs/reference/project-config

## Live Development

To run in live development mode, run `wails dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes. If you want to develop in a browser
and have access to your Go methods, there is also a dev server that runs on http://localhost:34115. Connect
to this in your browser, and you can call your Go code from devtools.

## Building

To build a redistributable, production mode package, use `wails build`.

# Application Updater

A desktop application for managing and updating multiple devices via HTTP.

## Features

### Device Management

- Manage device lists using local files
- Automatically search for devices by IP address range
- Test device connectivity via HTTP
- Manual device addition with connectivity testing
- Refresh device status
- Remove devices from the list

### Program Updates

- Login to devices with username/password
- Update devices by uploading files via HTTP
- View update operation results for each device

## Development

### Prerequisites

- Go 1.18 or later
- Node.js 14+ and npm
- Wails CLI v2

### Installation

1. Install Wails CLI if you haven't already:

   ```
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```

2. Clone this repository:

   ```
   git clone [repository-url]
   cd application-updater
   ```

3. Install dependencies:
   ```
   wails dev
   ```

### Development Mode

Run the application in development mode:

```
wails dev
```

This will start the application with hot-reload for both frontend and backend.

### Building

Build the application:

```
wails build
```

This will create a production-ready binary in the `build/bin` directory.

## Usage

1. **Device Management Tab**:

   - Add devices manually by entering their IP addresses
   - Scan IP ranges to automatically discover devices
   - View and refresh device statuses
   - Remove devices from the list

2. **Update Devices Tab**:
   - Enter the common username and password for all devices
   - Select the update file to upload
   - Click "Update All Devices" to start the update process
   - View the results for each device

## License

[MIT License](LICENSE)
# application-updater
