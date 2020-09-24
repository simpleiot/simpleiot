module Api.User exposing
    ( User
    , decode
    , decodeList
    , delete
    , empty
    , encode
    , findUser
    , getByEmail
    , list
    , update
    )

import Api.Data exposing (Data)
import Api.Response as Response exposing (Response)
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (required)
import Json.Encode as Encode
import List.Extra
import Url.Builder


type alias User =
    { id : String
    , first : String
    , last : String
    , phone : String
    , email : String
    , pass : String
    }


empty : User
empty =
    { id = ""
    , email = ""
    , pass = ""
    , first = ""
    , last = ""
    , phone = ""
    }


decodeList : Decode.Decoder (List User)
decodeList =
    Decode.list decode


decode : Decode.Decoder User
decode =
    Decode.succeed User
        |> required "id" Decode.string
        |> required "firstName" Decode.string
        |> required "lastName" Decode.string
        |> required "phone" Decode.string
        |> required "email" Decode.string
        |> required "pass" Decode.string


encode : User -> Encode.Value
encode user =
    Encode.object
        [ ( "firstName", Encode.string user.first )
        , ( "lastName", Encode.string user.last )
        , ( "phone", Encode.string user.phone )
        , ( "email", Encode.string user.email )
        , ( "pass", Encode.string user.pass )
        ]


findUser : List User -> String -> Maybe User
findUser users id =
    List.Extra.find (\u -> u.id == id) users


list :
    { token : String
    , onResponse : Data (List User) -> msg
    }
    -> Cmd msg
list options =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "users" ] []
        , expect = Api.Data.expectJson options.onResponse decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


delete :
    { token : String
    , id : String
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
delete options =
    Http.request
        { method = "DELETE"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "users", options.id ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


getByEmail :
    { token : String
    , email : String
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
getByEmail options =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "users" ] [ Url.Builder.string "email" options.email ]
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


update :
    { token : String
    , user : User
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
update options =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "users", options.user.id ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = options.user |> encode |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }
