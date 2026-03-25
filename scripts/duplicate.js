import { faker } from '@faker-js/faker';

fetch("http://localhost:9999/pessoas", {
    method: "POST",
    body: JSON.stringify({
        nome: faker.person.firstName(),
        apelido: "nickname",
        nascimento: "2003-07-19",
        stack: ["go", "js"]
    })
});

fetch("http://localhost:9999/pessoas", {
    method: "POST",
    body: JSON.stringify({
        nome: faker.person.firstName(),
        apelido: "nickname",
        nascimento: "2003-07-19",
        stack: ["go", "js"]
    })
});