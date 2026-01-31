# Changelog

All notable changes to ChameleonDB will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- LALRPOP-based schema parser with complete DSL support
- AST representation for entities, fields, and relations
- Support for field types: uuid, string, int, decimal, bool, timestamp
- Field modifiers: primary, unique, nullable, default
- Relation types: HasOne, HasMany, BelongsTo (ManyToMany pending)
- Serde serialization for all AST types

### In Progress
- FFI layer for Rust ↔ Go integration
- Type checker for schema validation
- Go runtime with query executor

## [0.1.0] - TBD

Initial MVP release.