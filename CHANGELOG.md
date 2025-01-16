# Change Log
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).


## [Unreleased]

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