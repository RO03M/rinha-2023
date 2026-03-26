import { faker } from '@faker-js/faker';

for (let i = 0; i < 1; i++) {
    fetch("http://localhost:9999/pessoas", {
        method: "POST",
        body: JSON.stringify({
            nome: faker.person.firstName(),
            apelido: faker.person.firstName(),
            nascimento: "1990-02-29",
            stack: ["go", "js"]
        })
    });
}