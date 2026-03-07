package com.example.inventory.api;

import java.util.Map;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class InventoryController {

    @GetMapping("/healthz")
    public Map<String, String> health() {
        return Map.of("status", "ok", "service", "inventory-api");
    }
}
