@RestController
public class Controller {

    @GetMapping("/index")
    public String index() {
        return "Hello Docker!";
    }

}