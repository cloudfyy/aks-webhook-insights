package com.example.demo;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@RestController
public class Controller {

    @GetMapping("/index")
    public String index() {
        return "Hello Docker!";
    }

}