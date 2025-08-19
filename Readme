### Application Flow

- The user sends their phone number to `/login` via a POST request.
- If the phone number is valid and no OTP code is currently active for that number, the server responds with a **201 status code**.
- The user must then send their phone number along with a valid OTP code with a POST request to `/check`. If the code is valid and the user hasn’t exceeded the rate limit (3 requests per 10 minutes), a JWT containing the user’s ID will be returned.
  You can also search for a user by phone number or retrieve a list of users by their registration date at `/search`. Requesting this path without any query will return the list of all users. The response can be customized using pagination settings.
  All documents are available via Swagger at `/swagger`.

### Database

Due to its high flexibility and speed, I chose MongoDB as the primary database. Being a document-based database, MongoDB provides an easy and fast environment for developing new staged applications.  
My reason for choosing MongoDB over other document-based databases is that it is very well-documented and has an active community, which is helpful when any trouble occurs.  
I avoided custom in-memory databases because they make further development harder and slower.
For saving OTP codes and implementing rate limiting, I used Redis. Speed-wise, an in-memory database is preferred, so I didn’t use MongoDB. Also, a custom in-memory database would slow down and complicate further development.

### How To Run

Everything is dockerized. Just clone the project with `git clone github.com/aph138/dekamond` and run `docker compose up -d` to start the application. For production use, make sure to set passwords for the databases and use `.env` file for environment variables.
