package com.example.inventory.service;

import org.springframework.stereotype.Service;

@Service
public class InventoryService {
    public String reservationMode() {
        return "strict";
    }
}
