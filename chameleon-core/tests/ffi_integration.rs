use std::ffi::{CStr, CString};
use std::ptr;

// Import desde el crate
use chameleon::{
    chameleon_parse_schema,
    chameleon_validate_schema,
    chameleon_free_string,
    chameleon_version,
    ChameleonResult,
};

#[test]
fn test_ffi_roundtrip() {
    let schema = r#"
        entity User {
            id: uuid primary,
            email: string unique,
            orders: [Order] via user_id,
        }
        entity Order {
            id: uuid primary,
            total: decimal,
            user_id: uuid,
            user: User,
        }
    "#;
    
    let input = CString::new(schema).unwrap();
    let mut __error__: *mut i8 = ptr::null_mut();
    
    unsafe {
        let result = chameleon_parse_schema(input.as_ptr(), &mut __error__);
        
        assert!(!result.is_null(), "Should parse successfully");
        assert!(__error__.is_null(), "Should have no errors");
        
        let json = CStr::from_ptr(result).to_str().unwrap();
        
        // Verify JSON structure
        assert!(json.contains("\"User\""));
        assert!(json.contains("\"Order\""));
        assert!(json.contains("\"email\""));
        
        println!("Parsed JSON:\n{}", json);
        
        // TODO(v1.0-stable): Fix chameleon_validate_schema to accept JSON
        // Currently it expects .cham syntax, not JSON
        // let validation_result = chameleon_validate_schema(result, &mut __error__);
        // assert_eq!(validation_result, ChameleonResult::Ok);
        
        chameleon_free_string(result);
    }
}

#[test]
fn test_version() {
    unsafe {
        let version = CStr::from_ptr(chameleon_version());
        let version_str = version.to_str().unwrap();
        assert_eq!(version_str, env!("CARGO_PKG_VERSION"));
        println!("ChameleonDB version: {}", version_str);
    }
}

#[test]
fn test_error_handling() {
    let input = CString::new("this is not valid syntax").unwrap();
    let mut __error__: *mut i8 = ptr::null_mut();
    
    unsafe {
        let result = chameleon_parse_schema(input.as_ptr(), &mut __error__);
        
        assert!(result.is_null(), "Should fail to parse");
        assert!(!__error__.is_null(), "Should have error message");
        
        let error_msg = CStr::from_ptr(__error__).to_str().unwrap();
        println!("Error message: {}", error_msg);
        
        // CAMBIO: Buscar "ParseError" sin espacio
        assert!(error_msg.contains("ParseError") || error_msg.contains("Parse error"));
        
        chameleon_free_string(__error__);
    }
}