# Change Log
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).


## [Unreleased]

## [v0.5.0-rc.1] - 2025-03-10
### Added
- Implement prometheus server on port :9191
- Add prometheus metrics to middleware: http_request_duration_seconds, http_requests_total

## [v0.4.0-rc.1] - 2025-03-10
### Added
- Add integration with rabbitmq with retries
- endpoints and dependent on it services/repositories: ScheduleTask


## [v0.3.0-rc.1] - 2025-01-16
### Added
- sample test for getting task by id
- entities: messenger, messengerRelatedUser
- endpoints and dependent on it services/repositories: CreateMessenger, GetMessenger, GetMessengerIDByName, CreateMessengerRelatedUser, GetMessengerRelatedUser, GetUserID (using messengerUserID)


## [v0.2.0-rc.1] - 2025-01-05
### Changed
- Add 'deleted_at' to predicates in queries
- Update dependencies

### Added
- LICENSE
- endpoints: DeleteTask, UpdateUser, DeleteUser

### Fixed
- update model's tags to hide some fields 


<!-- links -->
[v0.2.0-rc.1]: https://github.com/boskuv/goreminder/compare/v0.1.0-rc.1...v0.2.0-rc.1
[v0.3.0-rc.1]: https://github.com/boskuv/goreminder/compare/v0.2.0-rc.1...v0.3.0-rc.1
[v0.4.0-rc.1]: https://github.com/boskuv/goreminder/compare/v0.3.0-rc.1...v0.4.0-rc.1
[v0.5.0-rc.1]: https://github.com/boskuv/goreminder/compare/v0.4.0-rc.1...v0.5.0-rc.1