module Global exposing
    ( Auth(..)
    , Flags
    , Model
    , Msg(..)
    , Session
    , init
    , navigate
    , send
    , subscriptions
    , update
    , view
    )

import Browser.Navigation as Nav
import Data.Auth
import Data.Data as Data
import Data.Device as D
import Data.Group as G
import Data.Point as P
import Data.Response exposing (Response)
import Data.User as U
import Document exposing (Document)
import Element exposing (..)
import Generated.Route as Route exposing (Route)
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (optional, required)
import List.Extra
import Task
import Time
import UI.Form as Form
import UI.Style as Style
import Url exposing (Url)
import Url.Builder



-- INIT


type alias Flags =
    ()


type alias Model =
    { flags : Flags
    , url : Url
    , key : Nav.Key
    , auth : Auth
    }


type Auth
    = SignedOut (Maybe Http.Error)
    | SignedIn Session


type alias Session =
    { cred : Data.Auth.Cred
    , authToken : String
    , isRoot : Bool
    , data : Data.Data
    , error : Maybe Http.Error
    , respError : Maybe String
    , posting : Bool
    , newGroupUser : Maybe U.User
    , newGroupDevice : Maybe D.Device
    , errorDispCount : Int
    }


init : Flags -> Url -> Nav.Key -> ( Model, Cmd Msg )
init flags url key =
    ( Model
        flags
        url
        key
        (SignedOut Nothing)
    , Nav.pushUrl key (Route.toHref Route.SignIn)
    )



-- UPDATE


type Msg
    = Navigate Route
    | SignIn Data.Auth.Cred
    | AuthResponse Data.Auth.Cred (Result Http.Error Data.Auth.Response)
    | RequestGroups
    | RequestDevices
    | RequestUsers
    | DevicesResponse (Result Http.Error (List D.Device))
    | GroupsResponse (Result Http.Error (List G.Group))
    | UsersResponse (Result Http.Error (List U.User))
    | DeleteDevice String
    | DeleteDeviceResponse String (Result Http.Error Response)
    | DeviceCancelCmd String
    | DeviceCancelCmdResponse String (Result Http.Error D.DeviceCmd)
    | SignOut
    | Tick Time.Posix
    | UpdateDevicePoint String P.Point
    | UpdateDeviceGroups String (List String)
    | UpdateDeviceCmd String D.DeviceCmd
    | UpdateUser U.User
    | DeleteUser String
    | DeleteUserResponse String (Result Http.Error Response)
    | UpdateGroup G.Group
    | DeleteGroup String
    | DeleteGroupResponse String (Result Http.Error Response)
    | ConfigPosted String (Result Http.Error Response)
    | DeviceCmdPosted String (Result Http.Error Response)
    | DevicePointPosted String (Result Http.Error Response)
    | UserPosted String (Result Http.Error Response)
    | GroupPosted String (Result Http.Error Response)
    | CheckUser String
    | CheckUserResponse (Result Http.Error U.User)
    | CheckDevice String
    | CheckDeviceResponse (Result Http.Error D.Device)


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case model.auth of
        SignedOut _ ->
            case msg of
                Navigate route ->
                    ( model
                    , Nav.pushUrl model.key (Route.toHref route)
                    )

                SignIn cred ->
                    ( { model | auth = SignedOut Nothing }
                    , login cred
                    )

                AuthResponse cred (Ok resp) ->
                    ( { model
                        | auth =
                            SignedIn
                                { authToken = resp.token
                                , isRoot = resp.isRoot
                                , cred = cred
                                , data = Data.empty
                                , error = Nothing
                                , respError = Nothing
                                , posting = False
                                , newGroupUser = Nothing
                                , newGroupDevice = Nothing
                                , errorDispCount = 0
                                }
                      }
                    , Nav.pushUrl model.key (Route.toHref Route.Top)
                    )

                AuthResponse _ (Err error) ->
                    ( { model | auth = SignedOut (Just error) }, Cmd.none )

                _ ->
                    ( model
                    , Cmd.none
                    )

        SignedIn sess ->
            let
                data =
                    sess.data
            in
            case msg of
                Navigate route ->
                    ( model
                    , Nav.pushUrl model.key (Route.toHref route)
                    )

                SignIn _ ->
                    ( model, Cmd.none )

                SignOut ->
                    ( { model | auth = SignedOut Nothing }
                    , Nav.pushUrl model.key (Route.toHref Route.SignIn)
                    )

                AuthResponse _ (Ok _) ->
                    ( model, Cmd.none )

                AuthResponse _ (Err err) ->
                    ( { model | auth = SignedOut (Just err) }
                    , Nav.pushUrl model.key (Route.toHref Route.SignIn)
                    )

                DevicesResponse (Ok devices) ->
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | data = { data | devices = devices }
                                }
                      }
                    , Cmd.none
                    )

                DevicesResponse (Err err) ->
                    let
                        signOut =
                            case err of
                                Http.BadStatus code ->
                                    code == 401

                                _ ->
                                    False
                    in
                    if signOut then
                        ( { model | auth = SignedOut Nothing }
                        , Nav.pushUrl model.key (Route.toHref Route.SignIn)
                        )

                    else
                        ( { model
                            | auth =
                                SignedIn
                                    { sess
                                        | respError = Just ("Error getting devices: " ++ errorToString err)
                                        , errorDispCount = 0
                                    }
                          }
                        , Cmd.none
                        )

                UsersResponse (Ok users) ->
                    ( { model
                        | auth = SignedIn { sess | data = { data | users = users } }
                      }
                    , Cmd.none
                    )

                UsersResponse (Err err) ->
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | respError = Just ("Error getting users: " ++ errorToString err)
                                    , errorDispCount = 0
                                }
                      }
                    , Cmd.none
                    )

                RequestDevices ->
                    ( model
                    , if sess.posting then
                        Cmd.none

                      else
                        getDevices sess.authToken
                    )

                RequestUsers ->
                    ( model
                    , getUsers sess.authToken
                    )

                GroupsResponse (Ok groups) ->
                    ( { model
                        | auth = SignedIn { sess | data = { data | groups = groups } }
                      }
                    , Cmd.none
                    )

                GroupsResponse (Err err) ->
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | respError = Just ("Error getting groups" ++ errorToString err)
                                    , errorDispCount = 0
                                }
                      }
                    , Cmd.none
                    )

                RequestGroups ->
                    ( model
                    , getGroups sess.authToken
                    )

                Tick _ ->
                    let
                        respError =
                            if sess.errorDispCount > 5 then
                                Nothing

                            else
                                sess.respError
                    in
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | errorDispCount = sess.errorDispCount + 1
                                    , respError = respError
                                }
                      }
                    , Cmd.none
                    )

                UpdateDevicePoint id point ->
                    let
                        devices =
                            List.map
                                (\d ->
                                    if d.id == id then
                                        { d | points = P.updatePoint d.points point }

                                    else
                                        d
                                )
                                data.devices
                    in
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | data = { data | devices = devices }
                                    , posting = True
                                }
                      }
                    , postDevicePoint sess.authToken id point
                    )

                UpdateDeviceGroups id groups ->
                    let
                        devices =
                            List.map
                                (\d ->
                                    if d.id == id then
                                        { d | groups = groups }

                                    else
                                        d
                                )
                                data.devices
                    in
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | data = { data | devices = devices }
                                    , posting = True
                                    , newGroupDevice = Nothing
                                }
                      }
                    , postDeviceGroups sess.authToken id groups
                    )

                UpdateDeviceCmd id cmd ->
                    ( model, postDeviceCmd sess.authToken id cmd )

                UpdateUser user ->
                    let
                        -- update local model to make UI optimistic
                        updateUser old =
                            if old.id == user.id then
                                user

                            else
                                old

                        users =
                            if user.id == "" then
                                user :: sess.data.users

                            else
                                List.map updateUser sess.data.users
                    in
                    ( { model
                        | auth = SignedIn { sess | data = { data | users = users } }
                      }
                    , postUser sess.authToken user
                    )

                UpdateGroup group ->
                    let
                        -- update local model to make UI optimistic
                        updateGroup old =
                            if old.id == group.id then
                                group

                            else
                                old

                        groups =
                            if group.id == "" then
                                group :: sess.data.groups

                            else
                                List.map updateGroup sess.data.groups
                    in
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | data = { data | groups = groups }
                                    , newGroupUser = Nothing
                                }
                      }
                    , postGroup sess.authToken group
                    )

                DeleteGroup id ->
                    let
                        groups =
                            List.filter (\o -> o.id /= id) data.groups
                    in
                    ( { model
                        | auth = SignedIn { sess | data = { data | groups = groups } }
                      }
                    , deleteGroup sess.authToken id
                    )

                DeleteGroupResponse _ (Ok _) ->
                    ( model
                    , Cmd.none
                    )

                DeleteGroupResponse _ (Err err) ->
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | respError = Just ("Error deleting group" ++ errorToString err)
                                    , posting = False
                                    , errorDispCount = 0
                                }
                      }
                    , Cmd.none
                    )

                DeleteDevice id ->
                    let
                        devices =
                            List.filter (\d -> d.id /= id) data.devices
                    in
                    ( { model | auth = SignedIn { sess | data = { data | devices = devices } } }
                    , deleteDevice sess.authToken id
                    )

                DeleteDeviceResponse _ (Ok _) ->
                    ( model
                    , Cmd.none
                    )

                DeleteDeviceResponse _ (Err err) ->
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | respError = Just ("Error deleting device" ++ errorToString err)
                                    , posting = False
                                    , errorDispCount = 0
                                }
                      }
                    , Cmd.none
                    )

                DeviceCancelCmd id ->
                    ( model, deviceGetCmd sess.authToken id )

                DeviceCancelCmdResponse _ (Ok _) ->
                    ( model
                    , getDevices sess.authToken
                    )

                DeviceCancelCmdResponse _ (Err err) ->
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | respError = Just ("Error cancelling command" ++ errorToString err)
                                    , posting = False
                                    , errorDispCount = 0
                                }
                      }
                    , Cmd.none
                    )

                DeviceCmdPosted _ (Ok _) ->
                    ( model
                    , getDevices sess.authToken
                    )

                DeviceCmdPosted _ (Err err) ->
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | respError = Just ("Error saving device cmd" ++ errorToString err)
                                    , posting = False
                                    , errorDispCount = 0
                                }
                      }
                    , Cmd.none
                    )

                DevicePointPosted _ (Ok _) ->
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | posting = False
                                    , errorDispCount = 0
                                }
                      }
                    , getDevices sess.authToken
                    )

                DevicePointPosted _ (Err err) ->
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | respError = Just ("Error saving device config" ++ errorToString err)
                                    , posting = False
                                    , errorDispCount = 0
                                }
                      }
                    , Cmd.none
                    )

                ConfigPosted _ (Ok _) ->
                    ( { model | auth = SignedIn { sess | posting = False } }
                    , Cmd.none
                    )

                ConfigPosted _ (Err err) ->
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | respError = Just ("Error saving device config" ++ errorToString err)
                                    , posting = False
                                    , errorDispCount = 0
                                }
                      }
                    , Cmd.none
                    )

                UserPosted _ (Ok resp) ->
                    -- populate the assigned ID in the new user
                    let
                        users =
                            List.map
                                (\u ->
                                    if u.id == "" then
                                        { u | id = resp.id }

                                    else
                                        u
                                )
                                data.users
                    in
                    ( { model | auth = SignedIn { sess | data = { data | users = users } } }
                    , Cmd.none
                    )

                UserPosted _ (Err err) ->
                    -- refresh users as the local users cache is now
                    -- stale
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | respError = Just ("Error saving user" ++ errorToString err)
                                    , errorDispCount = 0
                                }
                      }
                    , getUsers sess.authToken
                    )

                GroupPosted _ (Ok resp) ->
                    -- populate the assigned ID in the new group
                    let
                        groups =
                            List.map
                                (\o ->
                                    if o.id == "" then
                                        { o | id = resp.id }

                                    else
                                        o
                                )
                                data.groups
                    in
                    ( { model | auth = SignedIn { sess | data = { data | groups = groups } } }
                    , Cmd.none
                    )

                GroupPosted _ (Err err) ->
                    -- refresh the ids because the local group cache is
                    -- is not correct because save did not take
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | respError = Just ("Error saving group" ++ errorToString err)
                                    , errorDispCount = 0
                                }
                      }
                    , getGroups sess.authToken
                    )

                CheckUser userEmail ->
                    ( { model | auth = SignedIn { sess | newGroupUser = Nothing } }
                    , getUserByEmail sess.authToken userEmail
                    )

                CheckUserResponse (Err _) ->
                    ( model, Cmd.none )

                CheckUserResponse (Ok user) ->
                    ( { model | auth = SignedIn { sess | newGroupUser = Just user } }
                    , Cmd.none
                    )

                CheckDevice deviceId ->
                    ( { model | auth = SignedIn { sess | newGroupDevice = Nothing } }
                    , getDeviceById sess.authToken deviceId
                    )

                CheckDeviceResponse (Err _) ->
                    ( model, Cmd.none )

                CheckDeviceResponse (Ok device) ->
                    -- make sure new device is in our local cache
                    -- of devices so we can modify it if necessary
                    let
                        devices =
                            case
                                List.Extra.find (\d -> d.id == device.id)
                                    sess.data.devices
                            of
                                Just _ ->
                                    sess.data.devices

                                Nothing ->
                                    device :: sess.data.devices
                    in
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | newGroupDevice = Just device
                                    , data = { data | devices = devices }
                                }
                      }
                    , Cmd.none
                    )

                DeleteUser id ->
                    let
                        users =
                            List.filter (\u -> u.id /= id) data.users
                    in
                    ( { model | auth = SignedIn { sess | data = { data | users = users } } }
                    , deleteUser sess.authToken id
                    )

                DeleteUserResponse _ (Ok _) ->
                    ( model
                    , Cmd.none
                    )

                DeleteUserResponse _ (Err err) ->
                    ( { model
                        | auth =
                            SignedIn
                                { sess
                                    | respError = Just ("Error deleting user" ++ errorToString err)
                                    , posting = False
                                    , errorDispCount = 0
                                }
                      }
                    , getUsers sess.authToken
                    )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.batch
        [ Time.every 1000 Tick
        ]



-- VIEW


view :
    { page : Document msg
    , global : Model
    , toMsg : Msg -> msg
    }
    -> Document msg
view { page, global, toMsg } =
    { title = page.title
    , body =
        [ column [ spacing 32, padding 20, width (fill |> maximum 780), height fill, centerX ]
            [ navbar global toMsg
            , viewError global
            , column [ height fill ] page.body
            , footer
            ]
        ]
    }



-- COMMANDS


send : msg -> Cmd msg
send =
    Task.succeed >> Task.perform identity


navigate : Route -> Cmd Msg
navigate route =
    send (Navigate route)



-- HTTP api


login : Data.Auth.Cred -> Cmd Msg
login cred =
    Http.post
        { body =
            Http.multipartBody
                [ Http.stringPart "email" cred.email
                , Http.stringPart "password" cred.password
                ]
        , url = Url.Builder.absolute [ "v1", "auth" ] []
        , expect = Http.expectJson (AuthResponse cred) Data.Auth.decodeResponse
        }


getDevices : String -> Cmd Msg
getDevices token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.Builder.absolute [ "v1", "devices" ] []
        , expect = Http.expectJson DevicesResponse D.decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


getDeviceById : String -> String -> Cmd Msg
getDeviceById token id =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.Builder.absolute [ "v1", "devices", id ] []
        , expect = Http.expectJson CheckDeviceResponse D.decode
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


deleteDevice : String -> String -> Cmd Msg
deleteDevice token id =
    Http.request
        { method = "DELETE"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.Builder.absolute [ "v1", "devices", id ] []
        , expect = Http.expectJson (DeleteDeviceResponse id) responseDecoder
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


deviceGetCmd : String -> String -> Cmd Msg
deviceGetCmd token id =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.Builder.absolute [ "v1", "devices", id, "cmd" ] []
        , expect = Http.expectJson (DeviceCancelCmdResponse id) D.decodeCmd
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


responseDecoder : Decode.Decoder Response
responseDecoder =
    Decode.succeed Response
        |> required "success" Decode.bool
        |> optional "error" Decode.string ""
        |> optional "id" Decode.string ""


postDeviceGroups : String -> String -> List String -> Cmd Msg
postDeviceGroups token id groups =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.Builder.absolute [ "v1", "devices", id, "groups" ] []
        , expect = Http.expectJson (ConfigPosted id) responseDecoder
        , body = groups |> D.encodeGroups |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


postDeviceCmd : String -> String -> D.DeviceCmd -> Cmd Msg
postDeviceCmd token id cmd =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.Builder.absolute [ "v1", "devices", id, "cmd" ] []
        , expect = Http.expectJson (DeviceCmdPosted id) responseDecoder
        , body = cmd |> D.encodeDeviceCmd |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


postDevicePoint : String -> String -> P.Point -> Cmd Msg
postDevicePoint token id point =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.Builder.absolute [ "v1", "devices", id, "points" ] []
        , expect = Http.expectJson (DevicePointPosted id) responseDecoder
        , body = [ point ] |> P.encodeList |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


getGroups : String -> Cmd Msg
getGroups token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.Builder.absolute [ "v1", "groups" ] []
        , expect = Http.expectJson GroupsResponse G.decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


getUsers : String -> Cmd Msg
getUsers token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.Builder.absolute [ "v1", "users" ] []
        , expect = Http.expectJson UsersResponse U.decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


deleteUser : String -> String -> Cmd Msg
deleteUser token id =
    Http.request
        { method = "DELETE"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.Builder.absolute [ "v1", "users", id ] []
        , expect = Http.expectJson (DeleteUserResponse id) responseDecoder
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


getUserByEmail : String -> String -> Cmd Msg
getUserByEmail token email =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.Builder.absolute [ "v1", "users" ] [ Url.Builder.string "email" email ]
        , expect = Http.expectJson CheckUserResponse U.decode
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


postUser : String -> U.User -> Cmd Msg
postUser token user =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.Builder.absolute [ "v1", "users", user.id ] []
        , expect = Http.expectJson (UserPosted user.id) responseDecoder
        , body = user |> U.encode |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


postGroup : String -> G.Group -> Cmd Msg
postGroup token group =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.Builder.absolute [ "v1", "groups", group.id ] []
        , expect = Http.expectJson (GroupPosted group.id) responseDecoder
        , body = group |> G.encode |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


deleteGroup : String -> String -> Cmd Msg
deleteGroup token id =
    Http.request
        { method = "DELETE"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.Builder.absolute [ "v1", "groups", id ] []
        , expect = Http.expectJson (DeleteGroupResponse id) responseDecoder
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }



-- UI Stuff


navbar : Model -> (Msg -> msg) -> Element msg
navbar model toMsg =
    row [ width fill, spacing 20 ]
        (link
            ( "SIOT", Route.Top )
            :: (case model.auth of
                    SignedIn sess ->
                        if sess.isRoot then
                            [ link ( "users", Route.Users )
                            , link ( "groups", Route.Groups )
                            ]

                        else
                            [ Element.none ]

                    SignedOut _ ->
                        [ Element.none ]
               )
            ++ [ el [ alignRight ] <|
                    case model.auth of
                        SignedIn sess ->
                            Form.button
                                ("sign out " ++ sess.cred.email)
                                Style.colors.blue
                                (toMsg SignOut)

                        SignedOut _ ->
                            viewButtonLink ( "sign in", Route.SignIn )
               ]
        )


viewButtonLink : ( String, Route ) -> Element msg
viewButtonLink ( label, route ) =
    Element.link (Style.button Style.colors.blue)
        { label = text label
        , url = Route.toHref route
        }


link : ( String, Route ) -> Element msg
link ( label, route ) =
    Element.link Style.link
        { label = text label
        , url = Route.toHref route
        }


footer : Element msg
footer =
    row [] [ Element.none ]



-- STYLES


viewError : Model -> Element msg
viewError model =
    case model.auth of
        SignedOut Nothing ->
            none

        SignedOut (Just err) ->
            el Style.error (el [ centerX ] (text ("Sign in failed: " ++ errorToString err)))

        SignedIn sess ->
            case sess.respError of
                Nothing ->
                    none

                Just error ->
                    el Style.error (el [ centerX ] (text error))


errorToString : Http.Error -> String
errorToString err =
    case err of
        Http.BadUrl url ->
            "Malformed url: " ++ url

        Http.Timeout ->
            "Timeout exceeded"

        Http.NetworkError ->
            "Network error"

        Http.BadStatus resp ->
            "Bad status: " ++ String.fromInt resp

        Http.BadBody resp ->
            "Bad body: " ++ resp
