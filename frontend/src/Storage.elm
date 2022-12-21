port module Storage exposing
    ( Storage, load
    , signIn, signOut
    , fromJson
    )

{-|

@docs Storage, save, load
@docs signIn, signOut

-}

import Api.Auth as Auth exposing (User)
import Json.Decode as Json
import Json.Encode as Encode


type alias Storage =
    { user : Maybe User
    }


fromJson : Json.Value -> Storage
fromJson json =
    json
        |> Json.decodeValue decoder
        |> Result.withDefault init


init : Storage
init =
    { user = Nothing
    }


decoder : Json.Decoder Storage
decoder =
    Json.map Storage
        (Json.field "user" (Json.maybe Auth.decode))


save : Storage -> Json.Value
save storage =
    Encode.object
        [ ( "user"
          , storage.user
                |> Maybe.map Auth.encode
                |> Maybe.withDefault Encode.null
          )
        ]



-- UPDATING STORAGE


signIn : User -> Storage -> Cmd msg
signIn user storage =
    saveToLocalStorage { storage | user = Just user }


signOut : Storage -> Cmd msg
signOut storage =
    saveToLocalStorage { storage | user = Nothing }



-- PORTS


saveToLocalStorage : Storage -> Cmd msg
saveToLocalStorage =
    save >> save_


port save_ : Json.Value -> Cmd msg


load : (Storage -> msg) -> Sub msg
load fromStorage =
    load_ (fromJson >> fromStorage)


port load_ : (Json.Value -> msg) -> Sub msg
