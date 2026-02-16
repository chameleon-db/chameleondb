func TestSelectSpecificFields(t *testing.T) {
    ctx := context.Background()
    engine := setupTestEngine(t)
    defer engine.Close()

    // Insert test user
    _, err := engine.Insert("User").
        Set("id", uuid.New().String()).
        Set("email", "test@mail.com").
        Set("name", "Test User").
        Set("age", 25).
        Execute(ctx)
    require.NoError(t, err)

    // Query with Select
    result, err := engine.Query("User").
        Select("id", "name").  // ← Only these fields
        Execute(ctx)
    require.NoError(t, err)
    require.Len(t, result.Rows, 1)

    row := result.Rows[0]
    
    // Should have selected fields
    assert.Contains(t, row, "id")
    assert.Contains(t, row, "name")
    
    // Should NOT have unselected fields
    assert.NotContains(t, row, "email")
    assert.NotContains(t, row, "age")
}

func TestSelectAllByDefault(t *testing.T) {
    ctx := context.Background()
    engine := setupTestEngine(t)
    defer engine.Close()

    // Insert test user
    _, err := engine.Insert("User").
        Set("id", uuid.New().String()).
        Set("email", "test@mail.com").
        Set("name", "Test User").
        Execute(ctx)
    require.NoError(t, err)

    // Query WITHOUT Select - should get all fields
    result, err := engine.Query("User").Execute(ctx)
    require.NoError(t, err)
    require.Len(t, result.Rows, 1)

    row := result.Rows[0]
    
    // Should have ALL fields
    assert.Contains(t, row, "id")
    assert.Contains(t, row, "name")
    assert.Contains(t, row, "email")
}